package partials

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// NotifyPlaygroundForm holds playground form field values for redisplay.
type NotifyPlaygroundForm struct {
	Mode           string
	ChannelID      int64
	TemplateID     string
	CustomTemplate string
	Format         string
	PayloadJSON    string
	Priority       string
	URL            string
}

// NotifyPlaygroundResultParams holds preview/send outcome for the playground result panel.
type NotifyPlaygroundResultParams struct {
	Title   string
	Body    string
	Format  string
	Preview bool
	Success bool
	Error   string
}

// NotifyPlaygroundParams is the view model for the Notifications playground tab.
type NotifyPlaygroundParams struct {
	Channels  []model.NotifyChannel
	Templates []config.NotifyTemplate
	Form      NotifyPlaygroundForm
	Errors    map[string]string
	Result    *NotifyPlaygroundResultParams
}

func playgroundFieldError(errors map[string]string, field string) string {
	if errors == nil || errors[field] == "" {
		return ""
	}
	return "input-error"
}

func playgroundPayloadErrorClass(errMsg string) string {
	if errMsg == "" {
		return ""
	}
	return "input-error"
}

func playgroundAlpineData(mode string) string {
	if mode == "" {
		mode = "template"
	}
	return fmt.Sprintf("{ mode: %q }", mode)
}
