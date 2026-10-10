package mcp

import (
	"net"
	"net/url"
	"strings"
)

// IsSelfMCP reports whether rawURL would reach this process's /mcp endpoint.
func IsSelfMCP(rawURL, listen, apiPath string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if !isMCPPath(u.Path, apiPath) {
		return false
	}
	listenHost, listenPort, err := net.SplitHostPort(strings.TrimSpace(listen))
	if err != nil {
		return false
	}
	uPort := u.Port()
	if uPort == "" {
		if u.Scheme == "https" {
			uPort = "443"
		} else {
			uPort = "80"
		}
	}
	if uPort != listenPort {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if isLoopbackHost(host) {
		return true
	}
	if host == "host.docker.internal" {
		return true
	}
	listenHost = strings.ToLower(listenHost)
	if listenHost != "" && host == listenHost {
		return true
	}
	return isLocalInterfaceHost(host)
}

func isMCPPath(path, apiPath string) bool {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}
	want := "/mcp"
	apiPath = strings.TrimSpace(apiPath)
	if apiPath != "" && apiPath != "/" {
		want = strings.TrimSuffix(apiPath, "/") + "/mcp"
		if !strings.HasPrefix(want, "/") {
			want = "/" + want
		}
	}
	return path == want || path == "/mcp"
}

func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0", "[::1]":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLocalInterfaceHost(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	local := map[string]struct{}{}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		local[ipNet.IP.String()] = struct{}{}
	}
	for _, ip := range ips {
		if _, ok := local[ip.String()]; ok {
			return true
		}
	}
	return false
}
