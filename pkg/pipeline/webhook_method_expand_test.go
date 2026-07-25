package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/pipeline"
)

func TestExpandDefinitionsWebhookMethod(t *testing.T) {
	t.Parallel()
	yamlStr := `name: webhook-test
enabled: true
triggers:
  - type: webhook
    enabled: true
    webhook:
      path: /a
      method: GET
      auth:
        token: "123456"
        hmac_secret: ""
steps:
  - name: s1
    capability: example
    operation: echo
`
	ed, err := pipeline.ParseEditorYAML(yamlStr)
	require.NoError(t, err)
	require.NotNil(t, ed.Triggers[0].Webhook)
	t.Logf("parsed method=%q path=%q token=%q hmac=%q", ed.Triggers[0].Webhook.Method, ed.Triggers[0].Webhook.Path, ed.Triggers[0].Webhook.Auth.Token, ed.Triggers[0].Webhook.Auth.HMACSecret)
	defs := pipeline.ExpandDefinitions([]pipeline.EditorDefinition{*ed})
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Trigger.Webhook)
	assert.Equal(t, "GET", defs[0].Trigger.Webhook.Method)
	assert.Equal(t, "123456", defs[0].Trigger.Webhook.Auth.Token)
}
