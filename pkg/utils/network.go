package utils

import (
	"net"
	"net/http"
	"strings"
	"time"
)

func PortAvailable(port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	defer func() {
		_ = conn.Close()
	}()
	if conn != nil {
		return false
	}
	return true
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
