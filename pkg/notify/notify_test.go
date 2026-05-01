package notify

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchema_Valid(t *testing.T) {
	scheme, err := ParseSchema("slack://hooks.slack.com/services/xxx")
	require.NoError(t, err)
	assert.Equal(t, "slack", scheme)
}

func TestParseSchema_Discord(t *testing.T) {
	scheme, err := ParseSchema("discord-bot://webhook/xxx")
	require.NoError(t, err)
	assert.Equal(t, "discord-bot", scheme)
}

func TestParseSchema_PlainText(t *testing.T) {
	scheme, err := ParseSchema("plain text")
	require.NoError(t, err)
	assert.Equal(t, "", scheme)
}

func TestParseSchema_Empty(t *testing.T) {
	scheme, err := ParseSchema("")
	require.NoError(t, err)
	assert.Equal(t, "", scheme)
}

func TestParseSchema_HTTPS(t *testing.T) {
	scheme, err := ParseSchema("https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "https", scheme)
}

func TestParseTemplate_SingleTemplate(t *testing.T) {
	templates := []string{"slack://{channel}/{token}"}
	result, err := ParseTemplate("slack://general/abc123", templates)
	require.NoError(t, err)
	assert.Equal(t, "general", result["channel"])
	assert.Equal(t, "abc123", result["token"])
}

func TestParseTemplate_NoMatch(t *testing.T) {
	templates := []string{"slack://{channel}/{token}"}
	result, err := ParseTemplate("https://other.com/path", templates)
	require.NoError(t, err)
	assert.Equal(t, types.KV{}, result)
}

func TestParseTemplate_MultipleTemplates(t *testing.T) {
	templates := []string{
		"discord://{channel}/{token}",
		"slack://{channel}/{token}",
	}
	result, err := ParseTemplate("slack://general/abc123", templates)
	require.NoError(t, err)
	assert.Equal(t, "general", result["channel"])
}

func TestParseTemplate_EmptyTemplates(t *testing.T) {
	result, err := ParseTemplate("slack://general/abc123", nil)
	require.NoError(t, err)
	assert.Equal(t, types.KV{}, result)
}

func TestParseTemplate_EmptyInput(t *testing.T) {
	templates := []string{"slack://{channel}"}
	result, err := ParseTemplate("", templates)
	require.NoError(t, err)
	assert.Equal(t, types.KV{}, result)
}

func TestParseTemplate_DashedKeys(t *testing.T) {
	templates := []string{"pushover://{user_key}/{app_token}"}
	result, err := ParseTemplate("pushover://ukey123/atoken", templates)
	require.NoError(t, err)
	assert.Equal(t, "ukey123", result["user_key"])
	assert.Equal(t, "atoken", result["app_token"])
}

func TestPriorityConstants(t *testing.T) {
	assert.Equal(t, Priority(1), Low)
	assert.Equal(t, Priority(2), Moderate)
	assert.Equal(t, Priority(3), Normal)
	assert.Equal(t, Priority(4), High)
	assert.Equal(t, Priority(5), Emergency)
}

func TestMessageZeroValue(t *testing.T) {
	m := Message{}
	assert.Empty(t, m.Title)
	assert.Empty(t, m.Body)
	assert.Empty(t, m.Url)
	assert.Equal(t, Priority(0), m.Priority)
}
