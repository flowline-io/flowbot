// Package forge implements the software forge capability.
package forge

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Descriptor returns the hub capability descriptor for forge.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapForge,
		Backend:     backend,
		App:         app,
		Description: "Forge capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: ability.OpForgeGetUser, Description: "Get authenticated user", Scopes: []string{auth.ScopeServiceForgeRead}},
			{Name: ability.OpForgeGetRepo, Description: "Get a repository", Scopes: []string{auth.ScopeServiceForgeRead}},
			{Name: ability.OpForgeListIssues, Description: "List issues", Scopes: []string{auth.ScopeServiceForgeRead}},
			{Name: ability.OpForgeGetIssue, Description: "Get an issue", Scopes: []string{auth.ScopeServiceForgeRead}},
			{Name: ability.OpForgeGetCommitDiff, Description: "Get commit diff", Scopes: []string{auth.ScopeServiceForgeRead}},
			{Name: ability.OpForgeGetFileContent, Description: "Get file content", Scopes: []string{auth.ScopeServiceForgeRead}},
		},
		Events: []hub.EventDef{
			{Name: types.EventForgeIssueOpened, Description: "Fires when an issue is opened"},
			{Name: types.EventForgeIssueClosed, Description: "Fires when an issue is closed"},
			{Name: types.EventForgeIssueReopened, Description: "Fires when an issue is reopened"},
			{Name: types.EventForgeIssueEdited, Description: "Fires when an issue is edited"},
			{Name: types.EventForgePush, Description: "Fires when code is pushed"},
		},
	}
}

// RegisterService registers the forge capability with the hub registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("forge capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpForgeGetUser, invoker: invokeGetUser(svc)},
		{operation: ability.OpForgeGetRepo, invoker: invokeGetRepo(svc)},
		{operation: ability.OpForgeListIssues, invoker: invokeListIssues(svc)},
		{operation: ability.OpForgeGetIssue, invoker: invokeGetIssue(svc)},
		{operation: ability.OpForgeGetCommitDiff, invoker: invokeGetCommitDiff(svc)},
		{operation: ability.OpForgeGetFileContent, invoker: invokeGetFileContent(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapForge, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeGetUser(svc Service) ability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*ability.InvokeResult, error) {
		user, err := svc.GetUser(ctx)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: user, Text: user.UserName}, nil
	}
}

func invokeGetRepo(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := ability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetRepo(ctx, owner, repo)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: item.FullName}, nil
	}
}

func invokeListIssues(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		q := &ListIssuesQuery{Page: ability.PageRequestFromParams(params)}
		if state, ok := ability.StringParam(params, "state"); ok {
			q.State = state
		}
		result, err := svc.ListIssues(ctx, owner, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(ability.OpForgeListIssues, result), nil
	}
}

func invokeGetIssue(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := ability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		index, err := ability.RequiredInt64(params, "index")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetIssue(ctx, owner, repo, index)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: item, Text: fmt.Sprintf("#%d %s", item.Index, item.Title)}, nil
	}
}

func invokeGetCommitDiff(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := ability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		commitID, err := ability.RequiredString(params, "commit_id")
		if err != nil {
			return nil, err
		}
		diff, err := svc.GetCommitDiff(ctx, owner, repo, commitID)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: diff, Text: diff.CommitMessage}, nil
	}
}

func invokeGetFileContent(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := ability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		commitID, err := ability.RequiredString(params, "commit_id")
		if err != nil {
			return nil, err
		}
		filePath, err := ability.RequiredString(params, "file_path")
		if err != nil {
			return nil, err
		}
		lineStart, _ := ability.IntParam(params, "line_start")
		lineCount, _ := ability.IntParam(params, "line_count")
		content, err := svc.GetFileContent(ctx, owner, repo, commitID, filePath, lineStart, lineCount)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: string(content), Text: filePath}, nil
	}
}

func listInvokeResult(operation string, result *ability.ListResult[ability.ForgeIssue]) *ability.InvokeResult {
	if result == nil {
		result = &ability.ListResult[ability.ForgeIssue]{Items: []*ability.ForgeIssue{}, Page: &ability.PageInfo{}}
	}
	return &ability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
