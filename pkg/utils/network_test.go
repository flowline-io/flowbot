package utils

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPortAvailable tests the PortAvailable function
func TestPortAvailable(t *testing.T) {
	t.Parallel()
	// Start a test server on a random port
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	// Extract port from server URL
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	require.NoError(t, err, "Failed to extract port")

	tests := []struct {
		name string
		port string
		want bool
	}{
		{
			name: "occupied_port",
			port: port,
			want: false, // PortAvailable returns false when we can connect (but conn != nil)
		},
		{
			name: "non_existent_port",
			port: "99999", // Assuming this port is not in use
			want: false,   // PortAvailable returns false when connection fails
		},
		{
			name: "empty_port_string",
			port: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PortAvailable(tt.port)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNetListener tests the NetListener function
func TestNetListener(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		addr    string
		wantErr bool
		network string
	}{
		{
			name:    "tcp_address",
			addr:    "127.0.0.1:0", // Use port 0 to get any available port
			wantErr: false,
			network: "tcp",
		},
		{
			name:    "invalid_tcp_address",
			addr:    "invalid:addr:format",
			wantErr: true,
			network: "tcp",
		},
		{
			name:    "unix_socket_address",
			addr:    "unix:/tmp/flowbot-test.sock",
			wantErr: false,
			network: "unix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			listener, err := NetListener(tt.addr)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, listener, "NetListener() returned nil listener")

			addr := listener.Addr()
			assert.NotNil(t, addr, "NetListener() listener has nil address")
			listener.Close()
		})
	}
}

// TestIsUnixAddr tests the IsUnixAddr function
func TestIsUnixAddr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{
			name: "unix_socket_addr",
			addr: "unix:/run/flowbot.sock",
			want: true,
		},
		{
			name: "tcp_addr",
			addr: "127.0.0.1:8080",
			want: false,
		},
		{
			name: "localhost_addr",
			addr: "localhost:3000",
			want: false,
		},
		{
			name: "just_port",
			addr: ":8080",
			want: false,
		},
		{
			name: "unix_without_path",
			addr: "unix:",
			want: true,
		},
		{
			name: "empty_addr",
			addr: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsUnixAddr(tt.addr)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestIsRoutableIP tests the IsRoutableIP function
func TestIsRoutableIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ipStr string
		want  bool
	}{
		{
			name:  "public_ipv4",
			ipStr: "8.8.8.8",
			want:  true,
		},
		{
			name:  "private_ipv4_10",
			ipStr: "10.0.0.1",
			want:  false,
		},
		{
			name:  "private_ipv4_192",
			ipStr: "192.168.1.1",
			want:  false,
		},
		{
			name:  "private_ipv4_172",
			ipStr: "172.16.0.1",
			want:  false,
		},
		{
			name:  "loopback_ipv4",
			ipStr: "127.0.0.1",
			want:  false,
		},
		{
			name:  "loopback_ipv6",
			ipStr: "::1",
			want:  false,
		},
		{
			name:  "link_local_ipv4",
			ipStr: "169.254.1.1",
			want:  false,
		},
		{
			name:  "invalid_ip",
			ipStr: "invalid.ip.address",
			want:  false,
		},
		{
			name:  "empty_ip",
			ipStr: "",
			want:  false,
		},
		{
			name:  "public_ipv6",
			ipStr: "2001:4860:4860::8888",
			want:  true,
		},
		{
			name:  "private_ipv6_unique_local",
			ipStr: "fc00::1",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsRoutableIP(tt.ipStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGetRemoteAddr tests the GetRemoteAddr function
func TestGetRemoteAddr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		xForwardedFor  string
		remoteAddr     string
		expectedResult string
	}{
		{
			name:           "no_x_forwarded_for",
			xForwardedFor:  "",
			remoteAddr:     "192.168.1.100:12345",
			expectedResult: "192.168.1.100:12345",
		},
		{
			name:           "private_x_forwarded_for",
			xForwardedFor:  "192.168.1.50",
			remoteAddr:     "10.0.0.1:12345",
			expectedResult: "10.0.0.1:12345",
		},
		{
			name:           "public_x_forwarded_for",
			xForwardedFor:  "8.8.8.8",
			remoteAddr:     "10.0.0.1:12345",
			expectedResult: "8.8.8.8",
		},
		{
			name:           "invalid_x_forwarded_for",
			xForwardedFor:  "invalid.ip",
			remoteAddr:     "127.0.0.1:12345",
			expectedResult: "127.0.0.1:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a mock HTTP request
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}

			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			result := GetRemoteAddr(req)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
