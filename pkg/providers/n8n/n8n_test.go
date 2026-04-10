package n8n

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

func TestN8N_ListWorkflows(t *testing.T) {
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
		err := json.NewEncoder(w).Encode(workflows)
		require.NoError(t, err)
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
}

func TestN8N_ListWorkflows_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "invalid-key")
	_, err := client.ListWorkflows()

	assert.Error(t, err)
}

func TestN8N_GetWorkflow(t *testing.T) {
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
		err := json.NewEncoder(w).Encode(workflow)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	result, err := client.GetWorkflow("workflow-123")

	require.NoError(t, err)
	assert.Equal(t, "workflow-123", result.ID)
	assert.Equal(t, "My Workflow", result.Name)
	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "n8n-nodes-base.webhook", result.Nodes[0].Type)
}

func TestN8N_CreateWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/workflows", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var reqData Workflow
		err = json.Unmarshal(body, &reqData)
		require.NoError(t, err)
		assert.Equal(t, "New Workflow", reqData.Name)

		createdWorkflow := Workflow{
			ID:     "new-workflow-id",
			Name:   reqData.Name,
			Active: reqData.Active,
			Nodes:  reqData.Nodes,
		}
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(createdWorkflow)
		require.NoError(t, err)
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
}

func TestN8N_UpdateWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/workflows/workflow-123", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var reqData Workflow
		err = json.Unmarshal(body, &reqData)
		require.NoError(t, err)

		updatedWorkflow := Workflow{
			ID:     "workflow-123",
			Name:   reqData.Name,
			Active: reqData.Active,
		}
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(updatedWorkflow)
		require.NoError(t, err)
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
}

func TestN8N_DeleteWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/workflows/workflow-123", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.DeleteWorkflow("workflow-123")

	require.NoError(t, err)
}

func TestN8N_ActivateWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/workflows/workflow-123/activate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.ActivateWorkflow("workflow-123")

	require.NoError(t, err)
}

func TestN8N_DeactivateWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/workflows/workflow-123/deactivate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.DeactivateWorkflow("workflow-123")

	require.NoError(t, err)
}

func TestN8N_ExecuteWorkflow_WithWebhookPath(t *testing.T) {
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
			err := json.NewEncoder(w).Encode(workflow)
			require.NoError(t, err)
		case "/webhook/my-webhook-path":
			assert.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var data map[string]any
			err = json.Unmarshal(body, &data)
			require.NoError(t, err)
			assert.Equal(t, "test-value", data["key"])
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.ExecuteWorkflow("workflow-123", map[string]any{"key": "test-value"})

	require.NoError(t, err)
}

func TestN8N_ExecuteWorkflow_WithWebhookID(t *testing.T) {
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
			err := json.NewEncoder(w).Encode(workflow)
			require.NoError(t, err)
		case "/webhook/webhook-456":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.ExecuteWorkflow("workflow-123", nil)

	require.NoError(t, err)
}

func TestN8N_ExecuteWorkflow_NoWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		err := json.NewEncoder(w).Encode(workflow)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewN8N(server.URL, "test-api-key")
	err := client.ExecuteWorkflow("workflow-123", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have a webhook trigger node")
}

func TestN8N_GetClient_Disabled(t *testing.T) {
	client, err := GetClient()
	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestNewN8N(t *testing.T) {
	client := NewN8N("https://n8n.example.com", "my-api-key")
	assert.NotNil(t, client)
	assert.NotNil(t, client.c)
}
