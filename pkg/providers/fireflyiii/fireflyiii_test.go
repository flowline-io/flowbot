package fireflyiii

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireflyIII_About(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		response   Response
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful about retrieval",
			response: Response{
				Data: map[string]any{
					"version":     "6.0.0",
					"api_version": "2.0.0",
					"php_version": "8.1.0",
					"os":          "Linux",
					"driver":      "mysql",
				},
				Message:   "",
				Exception: "",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server error",
			response:   Response{},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/about", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Contains(t, r.Header.Get("Accept"), "application/json")
				w.Header().Set("Content-Type", "application/json")

				w.WriteHeader(tt.statusCode)
				sonic.ConfigDefault.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewFireflyIII(server.URL, "test-token")
			result, err := client.About()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "6.0.0", result.Version)
				assert.Equal(t, "2.0.0", result.ApiVersion)
				assert.Equal(t, "Linux", result.Os)
			}
		})
	}
}

func TestFireflyIII_CurrentUser(t *testing.T) {
	t.Parallel()
	t.Run("successful current user retrieval", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/about/user", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")

			response := Response{
				Data: map[string]any{
					"type": "users",
					"id":   "1",
					"attributes": map[string]any{
						"created_at":   time.Now().Format(time.RFC3339),
						"updated_at":   time.Now().Format(time.RFC3339),
						"email":        "user@example.com",
						"blocked":      false,
						"blocked_code": "",
						"role":         "owner",
					},
				},
				Message:   "",
				Exception: "",
			}
			w.WriteHeader(http.StatusOK)
			err := sonic.ConfigDefault.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewFireflyIII(server.URL, "test-token")
		result, err := client.CurrentUser()

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "1", result.Id)
		assert.Equal(t, "user@example.com", result.Attributes.Email)
		assert.Equal(t, "owner", result.Attributes.Role)
	})
}

func TestFireflyIII_CreateTransaction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		transaction Transaction
		statusCode  int
		wantErr     bool
	}{
		{
			name: "successful transaction creation",
			transaction: Transaction{
				ApplyRules:   true,
				FireWebhooks: true,
				Transactions: []TransactionRecord{
					{
						Type:            string(Withdrawal),
						Date:            time.Now().Format("2006-01-02"),
						Amount:          "100.50",
						Description:     "Test transaction",
						SourceName:      "Cash",
						DestinationName: "Grocery Store",
						CategoryName:    "Food",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "validation error",
			transaction: Transaction{
				Transactions: []TransactionRecord{
					{
						Type: "invalid",
					},
				},
			},
			statusCode: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/transactions", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")

				body, _ := io.ReadAll(r.Body)
				var reqData Transaction
				err := sonic.Unmarshal(body, &reqData)
				assert.NoError(t, err)
				assert.Equal(t, tt.transaction.ApplyRules, reqData.ApplyRules)
				assert.Len(t, reqData.Transactions, len(tt.transaction.Transactions))

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					response := Response{
						Data: map[string]any{
							"type": "transactions",
							"id":   "123",
							"attributes": map[string]any{
								"created_at":   time.Now().Format(time.RFC3339),
								"updated_at":   time.Now().Format(time.RFC3339),
								"user":         "1",
								"group_title":  "Test Group",
								"transactions": []map[string]any{},
							},
						},
						Message:   "",
						Exception: "",
					}
					sonic.ConfigDefault.NewEncoder(w).Encode(response)
				} else {
					sonic.ConfigDefault.NewEncoder(w).Encode(Response{
						Message:   "Validation error",
						Exception: "FireflyIII\\Api\\V1\\Controllers\\TransactionController@store",
					})
				}
			}))
			defer server.Close()

			client := NewFireflyIII(server.URL, "test-token")
			result, err := client.CreateTransaction(tt.transaction)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "123", result.Id)
			}
		})
	}
}

func TestConvertResponseData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		response   *Response
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful conversion",
			response: &Response{
				Data: map[string]any{
					"version":     "6.0.0",
					"api_version": "2.0.0",
				},
				Message:   "",
				Exception: "",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "nil response",
			response:   nil,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name: "non-200 status code",
			response: &Response{
				Data:      map[string]any{},
				Message:   "Not found",
				Exception: "Exception",
			},
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name: "nil data on 200",
			response: &Response{
				Data:      nil,
				Message:   "",
				Exception: "",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertResponseData[About](tt.response, tt.statusCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestNewFireflyIII(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantNil  bool
	}{
		{name: "constructor creates client", endpoint: "https://firefly.example.com", token: "my-token"},
		{name: "empty endpoint returns nil", endpoint: "", token: "my-token", wantNil: true},
		{name: "endpoint with empty token", endpoint: "https://firefly.example.com", token: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewFireflyIII(tt.endpoint, tt.token)
			if tt.wantNil {
				assert.Nil(t, client)
				return
			}
			require.NotNil(t, client)
			assert.NotNil(t, client.c)
		})
	}
}

func TestFireflyIII_AboutUnauthenticatedJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header.Get("Accept"), "application/json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"Unauthenticated.","exception":"AuthenticationException"}`)
	}))
	t.Cleanup(server.Close)

	client := NewFireflyIII(server.URL, "bad-token")
	_, err := client.About()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestFireflyIII_AboutLoginHTMLRedirect(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/about", func(w http.ResponseWriter, r *http.Request) {
		// Mimic Firefly when Accept is missing: redirect to login.
		if !strings.Contains(r.Header.Get("Accept"), "application/json") {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"Unauthenticated.","exception":"AuthenticationException"}`)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html><body>login</body></html>")
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewFireflyIII(server.URL, "bad-token")
	_, err := client.About()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
