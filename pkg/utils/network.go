package utils

import (
	"net"
	"time"
)

func PortAvailable(port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 500*time.Millisecond)
	if err != nil {
		return true
	}
	if conn != nil {
		return false
	}
	return true
}
