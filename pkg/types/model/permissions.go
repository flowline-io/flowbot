package model

import "github.com/flowline-io/flowbot/pkg/agent/permission"

// PermissionsView is the narrow permission editor input for UI forms.
type PermissionsView struct {
	Defaults permission.Config `json:"defaults"`
	User     permission.Config `json:"user"`
}
