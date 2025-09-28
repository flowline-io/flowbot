package utils

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPortAvailable tests the PortAvailable function
func TestPortAvailable(t *testing.T) {
	// Start a test server on a random port
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port from server URL
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to extract port: %v", err)
	}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PortAvailable(tt.port)
			if got != tt.want {
				t.Errorf("PortAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNetListener tests the NetListener function
func TestNetListener(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := NetListener(tt.addr)

			if (err != nil) != tt.wantErr {
				t.Errorf("NetListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if listener == nil {
					t.Error("NetListener() returned nil listener")
				} else {
					// Check that we can get the address
					addr := listener.Addr()
					if addr == nil {
						t.Error("NetListener() listener has nil address")
					}
					listener.Close()
				}
			}
		})
	}
}

// TestIsUnixAddr tests the IsUnixAddr function
func TestIsUnixAddr(t *testing.T) {
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
			if got := IsUnixAddr(tt.addr); got != tt.want {
				t.Errorf("IsUnixAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsRoutableIP tests the IsRoutableIP function
func TestIsRoutableIP(t *testing.T) {
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
			if got := IsRoutableIP(tt.ipStr); got != tt.want {
				t.Errorf("IsRoutableIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetRemoteAddr tests the GetRemoteAddr function
func TestGetRemoteAddr(t *testing.T) {
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
			// Create a mock HTTP request
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}

			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			result := GetRemoteAddr(req)
			if result != tt.expectedResult {
				t.Errorf("GetRemoteAddr() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
