package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestHeaderTokenIgnoresEmptyBearer(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "abc", headerToken("abc", "Bearer ignored"))
	assert.Equal(t, "tok", headerToken("", "Bearer tok"))
	assert.Empty(t, headerToken("", ""))
	assert.Empty(t, headerToken("", "Basic x"))
}

func TestParseScopes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"pipeline:run", "hub:apps:read"}, parseScopes(types.KV{
		"scopes": []any{"pipeline:run", "hub:apps:read", ""},
	}))
	assert.Equal(t, []string{"admin:*"}, parseScopes(types.KV{"scopes": []string{"admin:*"}}))
	assert.Nil(t, parseScopes(types.KV{}))
	assert.Nil(t, parseScopes(types.KV{"scopes": "pipeline:run"}))
}
