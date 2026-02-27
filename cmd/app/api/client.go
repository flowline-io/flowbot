// Package api provides a unified HTTP client for the Admin frontend.
// All API calls go through this package, which automatically attaches the
// Authorization header. This package runs in a Wasm environment where
// net/http (and resty) uses the browser Fetch API under the hood.
package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"resty.dev/v3"
)

// defaultBasePath is the fallback API route prefix when no environment
// variable is provided.
const defaultBasePath = "/service/admin"

// clientOnce ensures the resty client is initialised exactly once,
// after the go-app environment is available.
var clientOnce sync.Once

// httpClient is the lazily-initialised resty HTTP client for all Admin API requests.
var httpClient *resty.Client

// getClient returns the shared resty client, initialising it on the first call
// using the API_BASE_URL environment variable injected by the PWA server.
func getClient() *resty.Client {
	clientOnce.Do(func() {
		baseURL := app.Getenv("API_BASE_URL")
		if baseURL == "" {
			baseURL = defaultBasePath
		}
		httpClient = resty.New().SetBaseURL(baseURL)
	})
	return httpClient
}

// APIResponse is the unified backend API response structure (aligned with protocol.Response).
type APIResponse struct {
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data,omitempty"`
	RetCode string          `json:"retcode,omitempty"`
	Message string          `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// Low-level request helper
// ---------------------------------------------------------------------------

// doRequest executes an HTTP request, automatically attaching the token and Content-Type.
func doRequest(token, method, path string, body interface{}) (json.RawMessage, error) {
	r := getClient().R().SetHeader("Content-Type", "application/json")

	if token != "" {
		r.SetHeader("Authorization", "Bearer "+token)
	}
	if body != nil {
		r.SetBody(body)
	}

	var apiResp APIResponse
	r.SetResult(&apiResp)

	var err error
	switch method {
	case "GET":
		_, err = r.Get(path)
	case "POST":
		_, err = r.Post(path)
	case "PUT":
		_, err = r.Put(path)
	case "DELETE":
		_, err = r.Delete(path)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("API error: %s (code: %s)", apiResp.Message, apiResp.RetCode)
	}

	return apiResp.Data, nil
}

// ---------------------------------------------------------------------------
// Authentication API
// ---------------------------------------------------------------------------

// GetSlackOAuthURL retrieves the Slack OAuth authorization URL.
func GetSlackOAuthURL(token string) (string, error) {
	data, err := doRequest(token, "GET", "/auth/slack/url", nil)
	if err != nil {
		return "", err
	}
	var resp admin.SlackOAuthURLResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	return resp.URL, nil
}

// DevLogin performs a quick dev-mode login (no Slack required) and returns a token.
func DevLogin(token string) (string, error) {
	data, err := doRequest(token, "POST", "/auth/dev-login", nil)
	if err != nil {
		return "", err
	}
	var resp admin.TokenResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

// GetCurrentUser retrieves information about the currently logged-in user.
func GetCurrentUser(token string) (*admin.UserInfo, error) {
	data, err := doRequest(token, "GET", "/auth/me", nil)
	if err != nil {
		return nil, err
	}
	var user admin.UserInfo
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ---------------------------------------------------------------------------
// System settings API
// ---------------------------------------------------------------------------

// GetSettings retrieves the current system settings.
func GetSettings(token string) (*admin.Settings, error) {
	data, err := doRequest(token, "GET", "/settings", nil)
	if err != nil {
		return nil, err
	}
	var settings admin.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings saves updated system settings.
func UpdateSettings(token string, s admin.Settings) error {
	_, err := doRequest(token, "PUT", "/settings", s)
	return err
}

// ---------------------------------------------------------------------------
// Container management API
// ---------------------------------------------------------------------------

// ListContainers fetches a paginated, searchable, sortable list of containers.
func ListContainers(token string, page, pageSize int, search, sortBy string, sortDesc bool) (*admin.ListResponse[admin.Container], error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("page_size", strconv.Itoa(pageSize))
	if search != "" {
		params.Set("search", search)
	}
	if sortBy != "" {
		params.Set("sort_by", sortBy)
	}
	if sortDesc {
		params.Set("sort_desc", "true")
	}

	data, err := doRequest(token, "GET", "/containers?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp admin.ListResponse[admin.Container]
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateContainer creates a new container.
func CreateContainer(token string, req admin.ContainerCreateRequest) (*admin.Container, error) {
	data, err := doRequest(token, "POST", "/containers", req)
	if err != nil {
		return nil, err
	}
	var container admin.Container
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}
	return &container, nil
}

// GetContainer retrieves a single container by ID.
func GetContainer(token string, id int64) (*admin.Container, error) {
	data, err := doRequest(token, "GET", fmt.Sprintf("/containers/%d", id), nil)
	if err != nil {
		return nil, err
	}
	var container admin.Container
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}
	return &container, nil
}

// UpdateContainer updates an existing container.
func UpdateContainer(token string, id int64, req admin.ContainerUpdateRequest) (*admin.Container, error) {
	data, err := doRequest(token, "PUT", fmt.Sprintf("/containers/%d", id), req)
	if err != nil {
		return nil, err
	}
	var container admin.Container
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}
	return &container, nil
}

// DeleteContainer removes a container by ID.
func DeleteContainer(token string, id int64) error {
	_, err := doRequest(token, "DELETE", fmt.Sprintf("/containers/%d", id), nil)
	return err
}

// BatchDeleteContainers removes multiple containers by their IDs.
func BatchDeleteContainers(token string, ids []int64) error {
	_, err := doRequest(token, "POST", "/containers/batch-delete", admin.BatchDeleteRequest{IDs: ids})
	return err
}
