package partials

import (
	"fmt"
	"net/url"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

// BlueprintWebPath is the catalog detail URL for a template.
func BlueprintWebPath(source, id string) string {
	return "/service/web/blueprints/" + url.PathEscape(source) + "/" + url.PathEscape(id)
}

// BlueprintInstantiatePath is the instantiate form URL.
func BlueprintInstantiatePath(source, id string) string {
	return BlueprintWebPath(source, id) + "/instantiate"
}

// BlueprintDownloadPath is the YAML download URL.
func BlueprintDownloadPath(source, id string) string {
	return BlueprintWebPath(source, id) + "/download"
}

// BlueprintListEntry is one catalog card.
type BlueprintListEntry struct {
	Source        string
	ID            string
	Title         string
	Description   string
	Missing       []string
	InstanceCount int
	Official      bool
}

// BlueprintInputView is one typed input on a blueprint form.
type BlueprintInputView struct {
	Name        string
	Type        string
	Required    bool
	Default     any
	Description string
}

// BlueprintDetailData is the read-only template view.
type BlueprintDetailData struct {
	Entry  BlueprintListEntry
	Inputs []BlueprintInputView
	YAML   string
}

// BlueprintInstantiateData is the instantiate form.
type BlueprintInstantiateData struct {
	Entry    BlueprintListEntry
	Inputs   []BlueprintInputView
	Channels []model.NotifyChannel
}

// BlueprintInstanceData is a linked pipeline page.
type BlueprintInstanceData struct {
	Pipeline        model.PipelineDefinition
	Title           string
	Inputs          []BlueprintInputView
	Channels        []model.NotifyChannel
	Enabled         bool
	UpdateAvailable bool
	Materialized    string
}

func notifyChannelChecked(values map[string]any, inputName, channel string) bool {
	if values == nil {
		return false
	}
	raw, ok := values[inputName]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case []string:
		for _, item := range v {
			if item == channel {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == channel {
				return true
			}
		}
	case string:
		return v == channel
	}
	return false
}

func boolInputChecked(values map[string]any, in BlueprintInputView) bool {
	if values != nil {
		if raw, ok := values[in.Name]; ok {
			if b, ok := raw.(bool); ok {
				return b
			}
		}
	}
	if b, ok := in.Default.(bool); ok {
		return b
	}
	return false
}

func stringInputValue(values map[string]any, in BlueprintInputView) string {
	if values != nil {
		if raw, ok := values[in.Name]; ok && raw != nil {
			return fmt.Sprint(raw)
		}
	}
	if in.Default != nil {
		return fmt.Sprint(in.Default)
	}
	return ""
}
