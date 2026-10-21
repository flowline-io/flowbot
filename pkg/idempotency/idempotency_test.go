package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestHash(t *testing.T) {
	t.Parallel()
	large, err := Hash(1e21)
	require.NoError(t, err)
	small, err := Hash(1e-7)
	require.NoError(t, err)
	nullHash, err := HashJSON([]byte("null"))
	require.NoError(t, err)
	blank, err := HashJSON(nil)
	require.NoError(t, err)
	empty, err := Hash(map[string]any{})
	require.NoError(t, err)
	ordered, err := Hash(map[string]any{"b": 1, "a": map[string]any{"d": true, "c": "x"}})
	require.NoError(t, err)
	reordered, err := HashJSON([]byte(`{"a":{"c":"x","d":true},"b":1.0}`))
	require.NoError(t, err)
	leftArr, err := Hash([]any{1, 2})
	require.NoError(t, err)
	rightArr, err := Hash([]any{2, 1})
	require.NoError(t, err)

	tests := []struct {
		name string
		got  string
		want string
		same bool
	}{
		{name: "object key order", got: ordered, want: reordered, same: true},
		{name: "blank matches empty object", got: blank, want: empty, same: true},
		{name: "null is not empty object", got: nullHash, want: empty, same: false},
		{name: "null canonical bytes", got: nullHash, want: digest("null"), same: true},
		{name: "large number uses exponent", got: large, want: digest("1e+21"), same: true},
		{name: "small number uses exponent", got: small, want: digest("1e-7"), same: true},
		{name: "array order", got: leftArr, want: rightArr, same: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.same {
				assert.Equal(t, tt.want, tt.got)
				return
			}
			assert.NotEqual(t, tt.want, tt.got)
		})
	}
}

func TestValidateKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "empty is absent", key: ""},
		{name: "allowed characters", key: "deploy_1.ok~now"},
		{name: "space", key: "has space", wantErr: true},
		{name: "too long", key: string(make([]byte, 256)), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		status   types.IdempotencyStatus
		stored   string
		want     string
		retained bool
		outcome  Outcome
	}{
		{name: "hash mismatch", status: types.IdempotencySucceeded, stored: "a", want: "b", retained: true, outcome: OutcomeMismatch},
		{name: "in progress", status: types.IdempotencyInProgress, stored: "a", want: "a", outcome: OutcomeInProgress},
		{name: "replay", status: types.IdempotencySucceeded, stored: "a", want: "a", retained: true, outcome: OutcomeReplay},
		{name: "success without body", status: types.IdempotencySucceeded, stored: "a", want: "a", outcome: OutcomeUnknown},
		{name: "unknown", status: types.IdempotencyUnknown, stored: "a", want: "a", outcome: OutcomeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.outcome, Decide(string(tt.status), tt.stored, tt.want, tt.retained))
		})
	}
}

func digest(canon string) string {
	sum := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(sum[:])
}
