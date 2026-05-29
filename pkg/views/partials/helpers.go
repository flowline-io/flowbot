// Package partials provides HTMX-targeted partial views.
package partials

import (
	"fmt"
	"net/url"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// valuePreview returns a truncated JSON representation of a KV map for display.
func valuePreview(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	s := string(b)
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

// fieldError returns a CSS border color class based on whether the field has a validation error.
func fieldError(errors map[string]string, field string) string {
	if _, ok := errors[field]; ok {
		return "border-red-500"
	}
	return "border-gray-300"
}

// valueJSON returns the full JSON string of a KV map.
func valueJSON(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// configKeyURL returns the key-based URL for a config item.
func configKeyURL(item model.ConfigItem) string {
	return fmt.Sprintf("/service/web/configs/%s/%s/%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configEditURL returns the edit URL for a config item.
func configEditURL(item model.ConfigItem) string {
	return configKeyURL(item) + "/edit"
}

// configRowID returns the DOM element ID for a config row.
func configRowID(item model.ConfigItem) string {
	return fmt.Sprintf("config-%s-%s-%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configFormID returns the DOM element ID for a config form row.
func configFormID(item model.ConfigItem, isNew bool) string {
	if isNew {
		return "config-form-new"
	}
	return "config-form-" + configRowID(item)
}

// cancelURL returns the cancel URL based on whether the form is for a new or existing item.
func cancelURL(item model.ConfigItem, isNew bool) string {
	if isNew {
		return "/service/web/configs/list"
	}
	return configKeyURL(item)
}
