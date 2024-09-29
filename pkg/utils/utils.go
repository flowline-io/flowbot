// Generic data manipulation utilities.

package utils

import (
	"crypto/tls"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/flowline-io/flowbot/pkg/config"
	"golang.org/x/crypto/acme/autocert"
)

const nullValue = "\u2421"

// stringDelta extracts the slices of added and removed strings from two slices:
//
//	added :=  newSlice - (oldSlice & newSlice) -- present in new but missing in old
//	removed := oldSlice - (oldSlice & newSlice) -- present in old but missing in new
//	intersection := oldSlice & newSlice -- present in both old and new
func stringSliceDelta(rold, rnew []string) (added, removed, intersection []string) {
	if len(rold) == 0 && len(rnew) == 0 {
		return nil, nil, nil
	}
	if len(rold) == 0 {
		return rnew, nil, nil
	}
	if len(rnew) == 0 {
		return nil, rold, nil
	}

	sort.Strings(rold)
	sort.Strings(rnew)

	// Match old slice against the new slice and separate removed strings from added.
	o, n := 0, 0
	lold, lnew := len(rold), len(rnew)
	for o < lold || n < lnew {
		if o == lold || (n < lnew && rold[o] > rnew[n]) {
			// Present in new, missing in old: added
			added = append(added, rnew[n])
			n++
		} else if n == lnew || rold[o] < rnew[n] {
			// Present in old, missing in new: removed
			removed = append(removed, rold[o])
			o++
		} else {
			// present in both
			intersection = append(intersection, rold[o])
			if o < lold {
				o++
			}
			if n < lnew {
				n++
			}
		}
	}
	return added, removed, intersection
}

// IsNullValue Check if the interface contains a string with a single Unicode Del control character.
func IsNullValue(i any) bool {
	if str, ok := i.(string); ok {
		return str == nullValue
	}
	return false
}

// ParseVersionPart Parse one component of a semantic version string.
func ParseVersionPart(vers string) int {
	end := strings.IndexFunc(vers, func(r rune) bool {
		return !unicode.IsDigit(r)
	})

	t := 0
	var err error
	if end > 0 {
		t, err = strconv.Atoi(vers[:end])
	} else if len(vers) > 0 {
		t, err = strconv.Atoi(vers)
	}
	if err != nil || t > 0x1fff || t <= 0 {
		return 0
	}
	return t
}

// ParseVersion Parses semantic version string in the following formats:
//
//	1.2, 1.2abc, 1.2.3, 1.2.3-abc, v0.12.34-rc5
//
// Unparceable values are replaced with zeros.
func ParseVersion(vers string) int {
	var major, minor, patch int
	// Maybe remove the optional "v" prefix.
	vers = strings.TrimPrefix(vers, "v")

	// We can handle 3 parts only.
	parts := strings.SplitN(vers, ".", 3)
	count := len(parts)
	if count > 0 {
		major = ParseVersionPart(parts[0])
		if count > 1 {
			minor = ParseVersionPart(parts[1])
			if count > 2 {
				patch = ParseVersionPart(parts[2])
			}
		}
	}

	return (major << 16) | (minor << 8) | patch
}

// Base10Version Version as a base-10 number. Used by monitoring.
func Base10Version(hex int) int64 {
	major := hex >> 16 & 0xFF
	minor := hex >> 8 & 0xFF
	trailer := hex & 0xFF
	return int64(major*10000 + minor*100 + trailer)
}

// VersionCompare Returns > 0 if v1 > v2; zero if equal; < 0 if v1 < v2
// Only Major and Minor parts are compared, the trailer is ignored.
func VersionCompare(v1, v2 int) int {
	return (v1 >> 8) - (v2 >> 8)
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ToAbsolutePath Convert relative filepath to absolute.
func ToAbsolutePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}

func ParseTLSConfig(tlsEnabled bool, conf config.TLSConfig) (*tls.Config, error) {
	if !tlsEnabled && !conf.Enabled {
		return nil, nil
	}

	// If autocert is provided, use it.
	if conf.Autocert != nil {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(conf.Autocert.Domains...),
			Cache:      autocert.DirCache(conf.Autocert.CertCache),
			Email:      conf.Autocert.Email,
		}
		return certManager.TLSConfig(), nil
	}

	// Otherwise try to use static keys.
	cert, err := tls.LoadX509KeyPair(conf.CertFile, conf.KeyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

// MergeMaps Deep copy maps.
func MergeMaps(dst, src map[string]any) (map[string]any, bool) {
	var changed bool

	if len(src) == 0 {
		return dst, changed
	}

	if dst == nil {
		dst = make(map[string]any)
	}

	for key, val := range src {
		xval := reflect.ValueOf(val)
		switch xval.Kind() {
		case reflect.Map:
			if xsrc, _ := val.(map[string]any); xsrc != nil {
				// Deep-copy map[string]interface{}
				xdst, _ := dst[key].(map[string]any)
				var lchange bool
				dst[key], lchange = MergeMaps(xdst, xsrc)
				changed = changed || lchange
			} else if val != nil {
				// The map is shallow-copied if it's not of the type map[string]interface{}
				dst[key] = val
				changed = true
			}
		case reflect.String:
			changed = true
			if xval.String() == nullValue {
				delete(dst, key)
			} else if val != nil {
				dst[key] = val
			}
		default:
			if val != nil {
				dst[key] = val
				changed = true
			}
		}
	}

	return dst, changed
}

// NetListener creates net.Listener for tcp and unix domains:
// if addr is in the form "unix:/run/flowbot.sock" it's a unix socket, otherwise TCP host:port.
func NetListener(addr string) (net.Listener, error) {
	addrParts := strings.SplitN(addr, ":", 2)
	if len(addrParts) == 2 && addrParts[0] == "unix" {
		return net.Listen("unix", addrParts[1])
	}
	return net.Listen("tcp", addr)
}

// IsUnixAddr Check if specified address is a unix socket like "unix:/run/flowbot.sock".
func IsUnixAddr(addr string) bool {
	addrParts := strings.SplitN(addr, ":", 2)
	return len(addrParts) == 2 && addrParts[0] == "unix"
}

var privateIPBlocks []*net.IPNet

func IsRoutableIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	if privateIPBlocks == nil {
		for _, cidr := range []string{
			"10.0.0.0/8",     // RFC1918
			"172.16.0.0/12",  // RFC1918
			"192.168.0.0/16", // RFC1918
			"fc00::/7",       // RFC4193, IPv6 unique local addr
		} {
			_, block, _ := net.ParseCIDR(cidr)
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

// GetRemoteAddr Obtain IP address of the client.
func GetRemoteAddr(req *http.Request) string {
	var addr string
	addr = req.Header.Get("X-Forwarded-For")
	if !IsRoutableIP(addr) {
		addr = ""
	}
	if addr != "" {
		return addr
	}
	return req.RemoteAddr
}
