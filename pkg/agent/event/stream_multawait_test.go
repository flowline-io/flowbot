package event_test

import (
	"context"
	"sync"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/stretchr/testify/assert"
)

func TestStream_MultipleAwait(t *testing.T) {
	tests := []struct {
		name     string
		waiters  int
		wantText string
	}{
		{name: "two waiters", waiters: 2, wantText: "done"},
		{name: "three waiters", waiters: 3, wantText: "done"},
		{name: "single waiter", waiters: 1, wantText: "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stream := event.NewStream(4)
			stream.End([]any{tt.wantText}, nil)

			var wg sync.WaitGroup
			wg.Add(tt.waiters)
			for range tt.waiters {
				go func() {
					defer wg.Done()
					result, err := stream.Await(context.Background())
					assert.NoError(t, err)
					if assert.Len(t, result.Messages, 1) {
						assert.Equal(t, tt.wantText, result.Messages[0])
					}
				}()
			}
			wg.Wait()
		})
	}
}
