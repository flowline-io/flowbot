package chatagent

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PermissionsView is the API payload for permission configuration.
type PermissionsView struct {
	Defaults      permission.Config   `json:"defaults"`
	User          permission.Config   `json:"user"`
	Effective     permission.Config   `json:"effective"`
	SessionGrants map[string][]string `json:"session_grants,omitempty"`
}

// BuildPermissionsView assembles permission state for one user and optional session.
func (s *Service) BuildPermissionsView(ctx context.Context, uid types.Uid, sessionID string) (PermissionsView, error) {
	user, err := loadUserPermissionConfig(ctx, uid)
	if err != nil {
		return PermissionsView{}, err
	}
	view := PermissionsView{
		Defaults:  permission.DefaultConfig(),
		User:      user,
		Effective: permission.EffectiveConfig(user),
	}
	if sessionID != "" {
		view.SessionGrants = s.permissionSessions.GetPermissionSession(ctx, sessionID).Grants()
	}
	return view, nil
}

// ClearSessionPermissionGrants resets always grants and doom-loop counters for one session.
func (s *Service) ClearSessionPermissionGrants(ctx context.Context, sessionID string) {
	state := s.permissionSessions.GetPermissionSession(ctx, sessionID)
	state.Clear()
	PersistSessionGrants(ctx, sessionID, state)
}

// ParsePermissionsBody unmarshals a PUT /chatagent/permissions request body.
func ParsePermissionsBody(raw []byte) (permission.Config, error) {
	cfg, err := permission.ParseConfig(raw)
	if err != nil {
		return nil, err
	}
	if err := permission.ValidateUserConfig(cfg); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return cfg, nil
}
