package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// AssertPublicHTTPURL validates that u is http(s) and does not target loopback,
// private, link-local, unspecified, or known cloud-metadata hostnames.
// When allowPrivate is true, only scheme/host presence are checked.
// Hostname literals that are not IPs are resolved via LookupIP before private checks.
func AssertPublicHTTPURL(u *url.URL, allowPrivate bool) error {
	if u == nil {
		return errors.New("empty url")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("only http and https URLs are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errors.New("url host is required")
	}
	if allowPrivate {
		return nil
	}
	if isBlockedHostname(host) {
		return fmt.Errorf("host %q is not allowed", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("address %s is not allowed", ip)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("host %q resolves to blocked address %s", host, ip)
		}
	}
	return nil
}

func isBlockedHostname(host string) bool {
	h := strings.ToLower(host)
	switch h {
	case "localhost", "metadata.google.internal":
		return true
	}
	return strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local")
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}
