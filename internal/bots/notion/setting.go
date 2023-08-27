package notion

import (
	"github.com/flowline-io/flowbot/internal/ruleset/setting"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	tokenSettingKey        = "token"
	importPageIdSettingKey = "import_page_id"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: tokenSettingKey, Type: types.FormFieldText, Title: "Internal Integration Token"},
	{Key: importPageIdSettingKey, Type: types.FormFieldText, Title: "MindCache page id"},
})
