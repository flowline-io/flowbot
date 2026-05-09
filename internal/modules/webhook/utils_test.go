package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/model"
)

func TestStateStr_Active(t *testing.T) {
	assert.Equal(t, "active", stateStr(model.WebhookActive))
}

func TestStateStr_Inactive(t *testing.T) {
	assert.Equal(t, "inactive", stateStr(model.WebhookInactive))
}

func TestStateStr_Unknown(t *testing.T) {
	assert.Equal(t, "unknown", stateStr(model.WebhookState(99)))
}
