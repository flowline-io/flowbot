package auth

import "github.com/flowline-io/flowbot/pkg/types"

type SubjectType string

const (
	SubjectUser     SubjectType = "user"
	SubjectToken    SubjectType = "token"
	SubjectWebhook  SubjectType = "webhook"
	SubjectCron     SubjectType = "cron"
	SubjectPipeline SubjectType = "pipeline"
	SubjectWorkflow SubjectType = "workflow"
	SubjectAgent    SubjectType = "agent"
)

type Context struct {
	SubjectType SubjectType `json:"subject_type"`
	SubjectID   string      `json:"subject_id"`
	UID         types.Uid   `json:"uid"`
	Topic       string      `json:"topic"`
	Scopes      []string    `json:"scopes"`
	IPAddress   string      `json:"ip_address,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
}

func (c Context) HasScope(scope string) bool {
	return HasScope(c.Scopes, scope)
}

func SystemCronContext() Context {
	return Context{SubjectType: SubjectCron, SubjectID: "system:cron"}
}

func SystemPipelineContext() Context {
	return Context{SubjectType: SubjectPipeline, SubjectID: "system:pipeline"}
}

func SystemWorkflowContext() Context {
	return Context{SubjectType: SubjectWorkflow, SubjectID: "system:workflow"}
}
