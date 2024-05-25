package webhook

import "github.com/flowline-io/flowbot/internal/store/model"

func stateStr(s model.WebhookState) string {
	switch s {
	case model.WebhookActive:
		return "active"
	case model.WebhookInactive:
		return "inactive"
	default:
		return "unknown"
	}
}
