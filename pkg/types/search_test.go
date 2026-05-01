package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentList_FillUrlBase(t *testing.T) {
	list := DocumentList{
		{Source: "github", Url: "/repos/foo"},
		{Source: "gitlab", Url: "/projects/1"},
		{Source: "unknown", Url: "/path"},
	}
	urlBase := map[string]string{
		"github": "https://github.com",
		"gitlab": "https://gitlab.com",
	}

	list.FillUrlBase(urlBase)

	assert.Equal(t, "https://github.com/repos/foo", list[0].Url)
	assert.Equal(t, "https://gitlab.com/projects/1", list[1].Url)
	assert.Equal(t, "/path", list[2].Url)
}

func TestDocumentList_FillUrlBase_Empty(t *testing.T) {
	var list DocumentList
	list.FillUrlBase(map[string]string{"x": "y"})
	assert.Nil(t, list)
}

func TestDocumentList_FillUrlBase_NoMatch(t *testing.T) {
	list := DocumentList{
		{Source: "bitbucket", Url: "/repo"},
	}
	list.FillUrlBase(map[string]string{"github": "https://github.com"})
	assert.Equal(t, "/repo", list[0].Url)
}

func TestRulesetTypeConstants(t *testing.T) {
	assert.Equal(t, RulesetType("action"), ActionRule)
	assert.Equal(t, RulesetType("command"), CommandRule)
	assert.Equal(t, RulesetType("cron"), CronRule)
	assert.Equal(t, RulesetType("event"), EventRule)
	assert.Equal(t, RulesetType("form"), FormRule)
	assert.Equal(t, RulesetType("page"), PageRule)
	assert.Equal(t, RulesetType("trigger"), TriggerRule)
	assert.Equal(t, RulesetType("webhook"), WebhookRule)
	assert.Equal(t, RulesetType("webservice"), WebserviceRule)
	assert.Equal(t, RulesetType("workflow"), WorkflowRule)
}
