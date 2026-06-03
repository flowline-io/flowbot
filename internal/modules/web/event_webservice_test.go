package web

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestHasWebhookData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		event *gen.DataEvent
		want  bool
	}{
		{
			name:  "nil data returns false",
			event: &gen.DataEvent{},
			want:  false,
		},
		{
			name: "has _webhook_method returns true",
			event: &gen.DataEvent{
				Data: map[string]any{"_webhook_method": "POST"},
			},
			want: true,
		},
		{
			name: "no webhook keys returns false",
			event: &gen.DataEvent{
				Data: map[string]any{"foo": "bar"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasWebhookData(tt.event))
		})
	}
}

func TestGetEventStore_Nil(t *testing.T) {
	t.Parallel()
	s := getEventStore()
	assert.Nil(t, s)
}

func TestParseEventFilterParams(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		assert func(t *testing.T, opts store.ListDataEventsOptions)
	}{
		{
			name:  "default values",
			query: "",
			assert: func(t *testing.T, opts store.ListDataEventsOptions) {
				assert.Equal(t, 20, opts.Limit)
				assert.Equal(t, 0, opts.Offset)
				assert.Empty(t, opts.Source)
				assert.Empty(t, opts.EventType)
				assert.Empty(t, opts.Search)
				assert.False(t, opts.Webhook)
			},
		},
		{
			name:  "all params set",
			query: "source=github&type=issue.created&search=bug&pipeline=deploy&page=2&per_page=10&time_start=2026-06-01T00:00:00Z&time_end=2026-06-03T00:00:00Z",
			assert: func(t *testing.T, opts store.ListDataEventsOptions) {
				assert.Equal(t, 10, opts.Limit)
				assert.Equal(t, 10, opts.Offset)
				assert.Equal(t, "github", opts.Source)
				assert.Equal(t, "issue.created", opts.EventType)
				assert.Equal(t, "bug", opts.Search)
				assert.Equal(t, "deploy", opts.PipelineName)
				assert.NotNil(t, opts.TimeStart)
				assert.NotNil(t, opts.TimeEnd)
			},
		},
		{
			name:  "per_page clamped to 100",
			query: "per_page=200",
			assert: func(t *testing.T, opts store.ListDataEventsOptions) {
				assert.Equal(t, 100, opts.Limit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)
			c.RequestCtx().Request.SetRequestURI("/events/filtered-events?" + tt.query)
			opts := parseEventFilterParams(c)
			tt.assert(t, opts)
		})
	}
}
