package chatagent

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ApprovalView is the API payload for approval mode configuration.
type ApprovalView struct {
	Mode           approval.Mode `json:"mode"`
	UserSet        bool          `json:"user_set"`
	ServerDefault  string        `json:"server_default"`
	ApprovalModel  string        `json:"approval_model"`
	EffectiveModel string        `json:"effective_model"`
}

// BuildApprovalView assembles approval mode state for one user.
func BuildApprovalView(ctx context.Context, uid types.Uid) (ApprovalView, error) {
	_, set, err := loadUserApprovalModeOverride(ctx, uid)
	if err != nil {
		return ApprovalView{}, err
	}
	effective, err := LoadUserApprovalMode(ctx, uid)
	if err != nil {
		return ApprovalView{}, err
	}
	return ApprovalView{
		Mode:           effective,
		UserSet:        set,
		ServerDefault:  config.ChatAgentApprovalModeDefault(),
		ApprovalModel:  config.App.ChatAgent.ApprovalModel,
		EffectiveModel: config.ResolveApprovalModel(),
	}, nil
}

// ParseApprovalMode parses a user-supplied approval mode string.
func ParseApprovalMode(raw string) (approval.Mode, error) {
	return approval.ParseMode(raw)
}

// ParseApprovalBody unmarshals a PUT /chatagent/approval request body.
func ParseApprovalBody(raw []byte) (approval.Mode, error) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := sonic.Unmarshal(raw, &body); err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "invalid approval json", err)
	}
	mode, err := approval.ParseMode(body.Mode)
	if err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "invalid approval mode", err)
	}
	return mode, nil
}
