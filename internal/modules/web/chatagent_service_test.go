package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestChatAgentServiceUnsetReturnsError(t *testing.T) {
	prev := webChatAgentService
	t.Cleanup(func() { webChatAgentService = prev })
	webChatAgentService = nil

	svc, err := chatAgentService()
	require.Error(t, err)
	assert.Nil(t, svc)
	require.ErrorIs(t, err, types.ErrInternal)
	assert.Equal(t, 0, pendingApprovalSessionCount())
}

func TestChatAgentServiceReturnsInstalled(t *testing.T) {
	prev := webChatAgentService
	t.Cleanup(func() { webChatAgentService = prev })
	ensureChatAgentService()

	svc, err := chatAgentService()
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Same(t, webChatAgentService, svc)
}
