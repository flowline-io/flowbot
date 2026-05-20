package n8n

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestN8N_ListWorkflows(t *testing.T) {
	t.Parallel()
	t.Run("successful list workflows", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "test-api-key", r.Header.Get("X-N8N-API-KEY"))
			w.Header().Set("Content-Type", "application/json")

			workflows := []*Workflow{
				{
					ID:        "1",
					Name:      "Test Workflow 1",
					Active:    true,
					Nodes:     []Node{},
					Tags:      []Tag{{ID: "tag1", Name: "test"}},
					CreatedAt: func() *time.Time { t := time.Now(); return &t }(),
				},
				{
					ID:        "2",
					Name:      "Test Workflow 2",
					Active:    false,
					Nodes:     []Node{},
					Tags:      []Tag{},
					CreatedAt: func() *time.Time { t := time.Now(); return &t }(),
				},
			}
			w.WriteHeader(http.StatusOK)
			err := sonic.ConfigDefault.NewEncoder(w).Encode(workflows)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		result, err := client.ListWorkflows()

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Test Workflow 1", result[0].Name)
		assert.True(t, result[0].Active)
		assert.Equal(t, "Test Workflow 2", result[1].Name)
		assert.False(t, result[1].Active)
	})
}

func TestN8N_ListWorkflows_Error(t *testing.T) {
	t.Parallel()
	t.Run("unauthorized error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "invalid-key")
		_, err := client.ListWorkflows()

		assert.Error(t, err)
	})
}

func TestN8N_GetWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful get workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows/workflow-123", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")

			workflow := Workflow{
				ID:     "workflow-123",
				Name:   "My Workflow",
				Active: true,
				Nodes: []Node{
					{
						ID:   "node-1",
						Name: "Webhook",
						Type: "n8n-nodes-base.webhook",
						Parameters: map[string]any{
							"path": "my-webhook",
						},
						WebhookID: "webhook-123",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			err := sonic.ConfigDefault.NewEncoder(w).Encode(workflow)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		result, err := client.GetWorkflow("workflow-123")

		require.NoError(t, err)
		assert.Equal(t, "workflow-123", result.ID)
		assert.Equal(t, "My Workflow", result.Name)
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "n8n-nodes-base.webhook", result.Nodes[0].Type)
	})
}

func TestN8N_CreateWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful create workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.Header().Set("Content-Type", "application/json")

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var reqData Workflow
			err = sonic.Unmarshal(body, &reqData)
			assert.NoError(t, err)
			assert.Equal(t, "New Workflow", reqData.Name)

			createdWorkflow := Workflow{
				ID:     "new-workflow-id",
				Name:   reqData.Name,
				Active: reqData.Active,
				Nodes:  reqData.Nodes,
			}
			w.WriteHeader(http.StatusCreated)
			err = sonic.ConfigDefault.NewEncoder(w).Encode(createdWorkflow)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		workflow := &Workflow{
			Name:   "New Workflow",
			Active: false,
			Nodes:  []Node{},
		}
		result, err := client.CreateWorkflow(workflow)

		require.NoError(t, err)
		assert.Equal(t, "new-workflow-id", result.ID)
		assert.Equal(t, "New Workflow", result.Name)
	})
}

func TestN8N_UpdateWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful update workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows/workflow-123", r.URL.Path)
			assert.Equal(t, http.MethodPut, r.Method)
			w.Header().Set("Content-Type", "application/json")

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var reqData Workflow
			err = sonic.Unmarshal(body, &reqData)
			assert.NoError(t, err)

			updatedWorkflow := Workflow{
				ID:     "workflow-123",
				Name:   reqData.Name,
				Active: reqData.Active,
			}
			w.WriteHeader(http.StatusOK)
			err = sonic.ConfigDefault.NewEncoder(w).Encode(updatedWorkflow)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		workflow := &Workflow{
			Name:   "Updated Workflow",
			Active: true,
		}
		result, err := client.UpdateWorkflow("workflow-123", workflow)

		require.NoError(t, err)
		assert.Equal(t, "Updated Workflow", result.Name)
		assert.True(t, result.Active)
	})
}

func TestN8N_DeleteWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful delete workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows/workflow-123", r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.DeleteWorkflow("workflow-123")

		require.NoError(t, err)
	})
}

func TestN8N_ActivateWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful activate workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows/workflow-123/activate", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.ActivateWorkflow("workflow-123")

		require.NoError(t, err)
	})
}

func TestN8N_DeactivateWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("successful deactivate workflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/workflows/workflow-123/deactivate", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.DeactivateWorkflow("workflow-123")

		require.NoError(t, err)
	})
}

func TestN8N_ExecuteWorkflow_WithWebhookPath(t *testing.T) {
	t.Parallel()
	t.Run("execute workflow with webhook path", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/workflows/workflow-123":
				workflow := Workflow{
					ID:     "workflow-123",
					Name:   "Test Workflow",
					Active: true,
					Nodes: []Node{
						{
							ID:   "node-1",
							Name: "Webhook",
							Type: "n8n-nodes-base.webhook",
							Parameters: map[string]any{
								"path": "my-webhook-path",
							},
							WebhookID: "webhook-123",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := sonic.ConfigDefault.NewEncoder(w).Encode(workflow)
				assert.NoError(t, err)
			case "/webhook/my-webhook-path":
				assert.Equal(t, http.MethodPost, r.Method)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				var data map[string]any
				err = sonic.Unmarshal(body, &data)
				assert.NoError(t, err)
				assert.Equal(t, "test-value", data["key"])
				w.WriteHeader(http.StatusOK)
			default:
				assert.Fail(t, "Unexpected request", "path", r.URL.Path)
			}
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.ExecuteWorkflow("workflow-123", map[string]any{"key": "test-value"})

		require.NoError(t, err)
	})
}

func TestN8N_ExecuteWorkflow_WithWebhookID(t *testing.T) {
	t.Parallel()
	t.Run("execute workflow with webhook ID", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/workflows/workflow-123":
				workflow := Workflow{
					ID:     "workflow-123",
					Name:   "Test Workflow",
					Active: true,
					Nodes: []Node{
						{
							ID:        "node-1",
							Name:      "Webhook",
							Type:      "n8n-nodes-base.webhookV2",
							WebhookID: "webhook-456",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := sonic.ConfigDefault.NewEncoder(w).Encode(workflow)
				assert.NoError(t, err)
			case "/webhook/webhook-456":
				w.WriteHeader(http.StatusAccepted)
			default:
				assert.Fail(t, "Unexpected request", "path", r.URL.Path)
			}
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.ExecuteWorkflow("workflow-123", nil)

		require.NoError(t, err)
	})
}

func TestN8N_ExecuteWorkflow_NoWebhook(t *testing.T) {
	t.Parallel()
	t.Run("execute workflow with no webhook", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			workflow := Workflow{
				ID:     "workflow-123",
				Name:   "Test Workflow",
				Active: true,
				Nodes: []Node{
					{
						ID:   "node-1",
						Name: "Manual Trigger",
						Type: "n8n-nodes-base.manualTrigger",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := sonic.ConfigDefault.NewEncoder(w).Encode(workflow)
			assert.NoError(t, err)
		}))
		defer server.Close()

		client := NewN8N(server.URL, "test-api-key")
		err := client.ExecuteWorkflow("workflow-123", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not have a webhook trigger node")
	})
}

func TestN8N_GetClient_Disabled(t *testing.T) {
	t.Parallel()
	t.Run("get client disabled", func(t *testing.T) {
		t.Parallel()
		client, err := GetClient()
		assert.Nil(t, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

func TestNewN8N(t *testing.T) {
	t.Parallel()
	t.Run("constructor creates client", func(t *testing.T) {
		t.Parallel()
		client := NewN8N("https://n8n.example.com", "my-api-key")
		assert.NotNil(t, client)
		assert.NotNil(t, client.c)
	})
}
