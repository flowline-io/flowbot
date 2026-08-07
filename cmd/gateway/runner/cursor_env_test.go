package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursorChildEnv(t *testing.T) {
	c := NewCursor("flowbot-agent", "cursor-key", 0).
		WithFlowbotAgent("http://127.0.0.1:6060", "agent-tok")
	env := c.childEnv()
	assert.Contains(t, env, "CURSOR_API_KEY=cursor-key")
	assert.Contains(t, env, "FLOWBOT_URL=http://127.0.0.1:6060")
	assert.Contains(t, env, "FLOWBOT_AGENT_TOKEN=agent-tok")
}
