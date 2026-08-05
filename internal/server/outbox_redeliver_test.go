package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

type stubOutboxStore struct {
	pending     []types.DataEvent
	listErr     error
	markErr     error
	markCalls   []string
	listCalls   int
	listOlder   time.Time
	listLimit   int
}

func (s *stubOutboxStore) ListPendingDataEventOutbox(_ context.Context, olderThan time.Time, limit int) ([]types.DataEvent, error) {
	s.listCalls++
	s.listOlder = olderThan
	s.listLimit = limit
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.pending, nil
}

func (s *stubOutboxStore) MarkOutboxPublished(_ context.Context, eventID string) error {
	s.markCalls = append(s.markCalls, eventID)
	return s.markErr
}

func TestRedeliverPendingOutbox(t *testing.T) {
	t.Parallel()
	pubErr := errors.New("publish failed")

	tests := []struct {
		name          string
		pending       []types.DataEvent
		listErr       error
		publishErr    error
		publishErrFor string
		markErr       error
		wantN         int
		wantErr       bool
		wantMarked    []string
		wantPublish   int
	}{
		{
			name: "empty pending",
		},
		{
			name: "publishes and marks each row",
			pending: []types.DataEvent{
				{EventID: "e1", EventType: "bookmark.created"},
				{EventID: "e2", EventType: "issue.created"},
			},
			wantN:       2,
			wantMarked:  []string{"e1", "e2"},
			wantPublish: 2,
		},
		{
			name:    "list error is returned",
			listErr: errors.New("db down"),
			wantErr: true,
		},
		{
			name: "publish failure skips mark and continues",
			pending: []types.DataEvent{
				{EventID: "bad", EventType: "x"},
				{EventID: "ok", EventType: "y"},
			},
			publishErrFor: "bad",
			wantN:         1,
			wantMarked:    []string{"ok"},
			wantPublish:   2,
		},
		{
			name: "mark failure does not count as published",
			pending: []types.DataEvent{
				{EventID: "e1", EventType: "bookmark.created"},
			},
			markErr:     errors.New("mark failed"),
			wantN:       0,
			wantMarked:  []string{"e1"},
			wantPublish: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &stubOutboxStore{pending: tt.pending, listErr: tt.listErr, markErr: tt.markErr}
			publishCount := 0
			publish := func(_ context.Context, topic string, payload any) error {
				publishCount++
				assert.Equal(t, DataEventTopic, topic)
				ev, ok := payload.(types.DataEvent)
				require.True(t, ok)
				if tt.publishErr != nil {
					return tt.publishErr
				}
				if tt.publishErrFor != "" && ev.EventID == tt.publishErrFor {
					return pubErr
				}
				return nil
			}
			n, err := redeliverPendingOutbox(context.Background(), s, publish, time.Now().Add(-time.Minute), 50)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantN, n)
			assert.Equal(t, tt.wantPublish, publishCount)
			assert.Equal(t, tt.wantMarked, s.markCalls)
		})
	}
}
