package trello

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	providers.Configs = []byte(`{"trello":{"webhook_token":"tok"}}`)
	w := NewWebhook()

	err := w.VerifySignature(map[string]string{"X-Query-Token": "tok"}, nil)
	require.NoError(t, err)

	err = w.VerifySignature(map[string]string{"X-Query-Token": "bad"}, nil)
	require.Error(t, err)
}

func TestWebhook_ConvertCreateCard(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	body := []byte(`{
		"action": {
			"id": "a1",
			"type": "createCard",
			"data": {"card": {"id": "c1", "name": "New"}}
		}
	}`)
	events, err := w.Convert(body, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, types.EventTrelloCardCreated, events[0].EventType)
	assert.Equal(t, "c1", events[0].EntityID)
	assert.Equal(t, "trello", events[0].Capability)
}

func TestWebhook_ConvertMoveCard(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	body := []byte(`{
		"action": {
			"id": "a2",
			"type": "updateCard",
			"data": {
				"card": {"id": "c2"},
				"listBefore": {"id": "l1"},
				"listAfter": {"id": "l2"}
			}
		}
	}`)
	events, err := w.Convert(body, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, types.EventTrelloCardMoved, events[0].EventType)
}

func TestWebhook_ConvertMoveCardListAfterOnly(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	body := []byte(`{
		"action": {
			"id": "a3",
			"type": "updateCard",
			"data": {
				"card": {"id": "c3"},
				"listAfter": {"id": "l2"}
			}
		}
	}`)
	events, err := w.Convert(body, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, types.EventTrelloCardMoved, events[0].EventType)
}

func TestMapTrelloActionUnsupported(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	events, err := w.Convert([]byte(`{"action":{"id":"a","type":"commentCard","data":{}}}`), nil)
	require.NoError(t, err)
	assert.Nil(t, events)
}
