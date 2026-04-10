package fireflyiii

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireflyIII_About(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/about", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
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
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewFireflyIII(server.URL, "test-token")
	result, err := client.CurrentUser()

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFireflyIII_CreateTransaction(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/transactions", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")

				body, _ := io.ReadAll(r.Body)
				var reqData Transaction
				err := json.Unmarshal(body, &reqData)
				require.NoError(t, err)
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
					json.NewEncoder(w).Encode(response)
				} else {
					json.NewEncoder(w).Encode(Response{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	client := NewFireflyIII("https://firefly.example.com", "my-token")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}
