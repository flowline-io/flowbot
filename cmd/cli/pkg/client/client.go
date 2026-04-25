// Package client provides an HTTP client for communicating with the Flowbot server.
package client

import (
	"encoding/json"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"resty.dev/v3"
)

// Client is an HTTP client for the Flowbot server API.
type Client struct {
	baseURL string
	rc      *resty.Client
}

// NewClient creates a new client with the given server URL and access token.
// The token is sent as the X-AccessToken header for authentication.
func NewClient(serverURL, token string) *Client {
	rc := resty.New()
	rc.SetBaseURL(serverURL)
	rc.SetHeader("X-AccessToken", token)
	rc.SetHeader("Content-Type", "application/json")

	return &Client{
		baseURL: serverURL,
		rc:      rc,
	}
}

// Response wraps the server's protocol.Response with the data parsed into the target type.
type Response struct {
	Status  string `json:"status"`
	RetCode string `json:"retcode,omitempty"`
	Message string `json:"message,omitempty"`
}

// Get performs a GET request and parses the response.
// The result should be a pointer to the target type for the Data field.
func (c *Client) Get(path string, result any) error {
	resp, err := c.rc.R().Get(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Post performs a POST request with the given body and parses the response.
func (c *Client) Post(path string, body any, result any) error {
	resp, err := c.rc.R().SetBody(body).Post(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Patch performs a PATCH request with the given body and parses the response.
func (c *Client) Patch(path string, body any, result any) error {
	resp, err := c.rc.R().SetBody(body).Patch(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Delete performs a DELETE request with the given body and parses the response.
func (c *Client) Delete(path string, body any, result any) error {
	resp, err := c.rc.R().SetBody(body).Delete(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// parseResponse unmarshals the HTTP response into a protocol.Response,
// checks for errors, and unmarshals the Data field into the target result.
func parseResponse(resp *resty.Response, result any) error {
	body := resp.Bytes()
	if len(body) == 0 {
		return fmt.Errorf("empty response")
	}

	var r protocol.Response
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if r.Status != protocol.Success {
		msg := r.Message
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("server error (%s): %s", r.RetCode, msg)
	}

	if result != nil && r.Data != nil {
		data, err := json.Marshal(r.Data)
		if err != nil {
			return fmt.Errorf("marshal response data: %w", err)
		}
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
	}

	return nil
}
