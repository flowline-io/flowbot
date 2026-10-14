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

func TestFireflyIII_ListTransactions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/transactions", r.URL.Path)
		assert.Equal(t, "withdrawal", r.URL.Query().Get("type"))
		assert.Equal(t, "2026-01-01", r.URL.Query().Get("start"))
		assert.Equal(t, "2026-01-31", r.URL.Query().Get("end"))
		assert.Equal(t, "5", r.URL.Query().Get("account_id"))
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{
			"data":[{"type":"transactions","id":"10","attributes":{"transactions":[{"type":"withdrawal","amount":"12.00","description":"Coffee","source_name":"Cash","destination_name":"Cafe","date":"2026-01-15T00:00:00+00:00"}]}}],
			"meta":{"pagination":{"total":1,"count":1,"per_page":25,"current_page":1,"total_pages":1}}
		}`)
	}))
	t.Cleanup(server.Close)

	client := NewFireflyIII(server.URL, "test-token")
	result, err := client.ListTransactions(TransactionListQuery{
		ListQuery: ListQuery{Page: 1, PerPage: 25},
		Type:      "withdrawal",
		Start:     "2026-01-01",
		End:       "2026-01-31",
		AccountID: "5",
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "10", result.Items[0].Id)
	assert.Equal(t, 1, result.Pagination.TotalPages)
}

func TestFireflyIII_GetUpdateDeleteTransaction(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transactions/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet, http.MethodPut:
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"data":{"type":"transactions","id":"42","attributes":{"transactions":[{"type":"withdrawal","amount":"9.00","description":"Updated","date":"2026-01-02T00:00:00+00:00"}]}}}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewFireflyIII(server.URL, "test-token")
	got, err := client.GetTransaction("42")
	require.NoError(t, err)
	assert.Equal(t, "42", got.Id)

	updated, err := client.UpdateTransaction("42", Transaction{
		Transactions: []TransactionRecord{{Type: "withdrawal", Amount: "9.00", Description: "Updated", Date: "2026-01-02"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "42", updated.Id)

	require.NoError(t, client.DeleteTransaction("42"))
}

func TestFireflyIII_ListResources(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	writeList := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}
	mux.HandleFunc("/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "asset", r.URL.Query().Get("type"))
		writeList(w, `{"data":[{"type":"accounts","id":"1","attributes":{"name":"Cash","type":"asset"}}],"meta":{"pagination":{"current_page":1,"total_pages":1,"per_page":50}}}`)
	})
	mux.HandleFunc("/v1/categories", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, `{"data":[{"type":"categories","id":"2","attributes":{"name":"Food"}}],"meta":{"pagination":{"current_page":1,"total_pages":1,"per_page":50}}}`)
	})
	mux.HandleFunc("/v1/budgets", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, `{"data":[{"type":"budgets","id":"3","attributes":{"name":"Groceries","active":true}}],"meta":{"pagination":{"current_page":1,"total_pages":1,"per_page":50}}}`)
	})
	mux.HandleFunc("/v1/bills", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, `{"data":[{"type":"bills","id":"4","attributes":{"name":"Rent","active":true}}],"meta":{"pagination":{"current_page":1,"total_pages":1,"per_page":50}}}`)
	})
	mux.HandleFunc("/v1/tags", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, `{"data":[{"type":"tags","id":"5","attributes":{"tag":"travel"}}],"meta":{"pagination":{"current_page":1,"total_pages":2,"per_page":50}}}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewFireflyIII(server.URL, "test-token")
	accounts, err := client.ListAccounts(AccountListQuery{Type: "asset"})
	require.NoError(t, err)
	require.Len(t, accounts.Items, 1)
	assert.Equal(t, "Cash", accounts.Items[0].Attributes.Name)

	categories, err := client.ListCategories(ListQuery{})
	require.NoError(t, err)
	assert.Equal(t, "Food", categories.Items[0].Attributes.Name)

	budgets, err := client.ListBudgets(ListQuery{})
	require.NoError(t, err)
	assert.True(t, budgets.Items[0].Attributes.Active)

	bills, err := client.ListBills(ListQuery{})
	require.NoError(t, err)
	assert.Equal(t, "Rent", bills.Items[0].Attributes.Name)

	tags, err := client.ListTags(ListQuery{})
	require.NoError(t, err)
	assert.Equal(t, "travel", tags.Items[0].Attributes.Tag)
	assert.Equal(t, 2, tags.Pagination.TotalPages)
}

func TestConvertListResponseData(t *testing.T) {
	t.Parallel()
	result, err := ConvertListResponseData[Account](&Response{
		Data: []any{map[string]any{"type": "accounts", "id": "1", "attributes": map[string]any{"name": "Cash", "type": "asset"}}},
		Meta: &Meta{Pagination: Pagination{CurrentPage: 1, TotalPages: 1}},
	}, http.StatusOK)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "1", result.Items[0].Id)

	_, err = ConvertListResponseData[Account](&Response{Data: nil}, http.StatusOK)
	assert.Error(t, err)
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
