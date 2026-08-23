package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantNil bool
	}{
		{name: "empty config returns nil", configs: json.RawMessage(`{}`), wantNil: true},
		{name: "api key only returns nil", configs: json.RawMessage(`{"trello":{"api_key":"key"}}`), wantNil: true},
		{name: "api key and token configured", configs: json.RawMessage(`{"trello":{"api_key":"key","token":"tok"}}`), wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c := GetClient()
			if tt.wantNil {
				assert.Nil(t, c)
				return
			}
			require.NotNil(t, c)
		})
	}
}

func TestListBoards(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/members/me/boards", r.URL.Path)
		assert.Equal(t, "key", r.URL.Query().Get("key"))
		assert.Equal(t, "token", r.URL.Query().Get("token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"b1","name":"Board One"}]`))
	}))
	defer srv.Close()

	client := NewTrello("key", "token")
	client.c.SetBaseURL(srv.URL)

	boards, err := client.ListBoards(context.Background())
	require.NoError(t, err)
	require.Len(t, boards, 1)
	assert.Equal(t, "b1", boards[0].ID)
	assert.Equal(t, "Board One", boards[0].Name)
}

func TestCreateCard(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/cards", r.URL.Path)
		assert.Equal(t, "list1", r.URL.Query().Get("idList"))
		assert.Equal(t, "Task", r.URL.Query().Get("name"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c1","name":"Task","idList":"list1","idBoard":"b1"}`))
	}))
	defer srv.Close()

	client := NewTrello("key", "token")
	client.c.SetBaseURL(srv.URL)

	card, err := client.CreateCard(context.Background(), "list1", "Task", "")
	require.NoError(t, err)
	assert.Equal(t, "c1", card.ID)
}

func TestGetWebhookToken(t *testing.T) {
	providers.Configs = json.RawMessage(`{"trello":{"webhook_token":"secret"}}`)
	assert.Equal(t, "secret", GetWebhookToken())
}
