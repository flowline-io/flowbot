package result_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

func TestIsContextOverflowErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not overflow",
			err:  nil,
			want: false,
		},
		{
			name: "typed overflow error",
			err:  result.NewOverflowError("context window exceeded", errors.New("upstream")),
			want: true,
		},
		{
			name: "prompt is too long message",
			err:  errors.New("prompt is too long"),
			want: true,
		},
		{
			name: "maximum context length message",
			err:  errors.New("This model's maximum context length is 128000 tokens"),
			want: true,
		},
		{
			name: "rate limit is not overflow",
			err:  errors.New("rate limit exceeded"),
			want: false,
		},
		{
			name: "throttling error is not overflow",
			err:  errors.New("Throttling error: TooManyRequests"),
			want: false,
		},
		{
			name: "unrelated error is not overflow",
			err:  errors.New("connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, result.IsContextOverflowErr(tt.err))
		})
	}
}

func TestWrapOverflowError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantSame bool
		wantCode string
		wantNil  bool
	}{
		{
			name:    "nil stays nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "non-overflow stays unchanged",
			err:      errors.New("connection refused"),
			wantSame: true,
		},
		{
			name:     "overflow wraps as OverflowError",
			err:      errors.New("token limit exceeded"),
			wantCode: "overflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := result.WrapOverflowError(tt.err)
			if tt.wantNil {
				assert.NoError(t, got)
				return
			}
			if tt.wantSame {
				assert.Equal(t, tt.err, got)
				return
			}
			require.Error(t, got)
			assert.Equal(t, tt.wantCode, result.CodeOf(got))
			assert.True(t, result.IsContextOverflowErr(got))
		})
	}
}

func TestMatchesOverflowText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "context length exceeded",
			text: "context_length_exceeded",
			want: true,
		},
		{
			name: "too many tokens",
			text: "too many tokens in request",
			want: true,
		},
		{
			name: "rate limit text is non-overflow",
			text: "rate limit hit",
			want: false,
		},
		{
			name: "empty text is not overflow",
			text: "",
			want: false,
		},
		{
			name: "unrelated text is not overflow",
			text: "internal server error",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, result.MatchesOverflowText(tt.text))
			if tt.name == "rate limit text is non-overflow" {
				assert.True(t, result.IsNonOverflowText(tt.text))
			}
		})
	}
}
