package confluence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	providers.Configs = []byte(`{"confluence":{"webhook_token":"tok"}}`)
	w := NewWebhook()
	require.NoError(t, w.VerifySignature(map[string]string{"X-Query-Token": "tok"}, nil))
	require.Error(t, w.VerifySignature(map[string]string{"X-Query-Token": "bad"}, nil))
}

func TestWebhook_ConvertPageCreated(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	body := []byte(`{"event":"page_created","id":"evt-1","page":{"id":"99","title":"Doc"}}`)
	events, err := w.Convert(body, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, types.EventConfluencePageCreated, events[0].EventType)
	assert.Equal(t, "99", events[0].EntityID)
	assert.Equal(t, "99:page_created:evt-1", events[0].IdempotencyKey)
}

func TestWebhook_ConvertPageCreatedAliasRejected(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	events, err := w.Convert([]byte(`{"event":"created","page":{"id":"1"}}`), nil)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestWebhook_ConvertUnsupported(t *testing.T) {
	t.Parallel()
	w := NewWebhook()
	events, err := w.Convert([]byte(`{"event":"comment_added"}`), nil)
	require.NoError(t, err)
	assert.Nil(t, events)
}
