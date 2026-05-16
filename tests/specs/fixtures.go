//go:build integration
// +build integration

package specs

import (
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/gomega"
)

// MakeRequest creates an HTTP request for testing.
func MakeRequest(method, path string, body []byte) *http.Request {
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}
	url := "http://localhost" + path
	req, err := http.NewRequest(method, url, bodyReader)
	Expect(err).NotTo(HaveOccurred())
	return req
}

// JSONRequest creates a JSON HTTP request for testing.
func JSONRequest(method, path string, body []byte) *http.Request {
	req := MakeRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ReadBody reads and returns the response body.
func ReadBody(resp *http.Response) []byte {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	return body
}
