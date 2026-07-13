// Package gitea implements the software forge capability.
package gitea

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the gitea capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapGitea,
		App:         app,
		Description: "Forge capability",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventForgeIssueOpened, Description: "Fires when an issue is opened"},
			{Name: types.EventForgeIssueClosed, Description: "Fires when an issue is closed"},
			{Name: types.EventForgeIssueReopened, Description: "Fires when an issue is reopened"},
			{Name: types.EventForgeIssueEdited, Description: "Fires when an issue is edited"},
			{Name: types.EventForgePush, Description: "Fires when code is pushed"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpGetUser, Description: "Get authenticated user", Scopes: []string{auth.ScopeServiceForgeRead},
				Handler: invokeGetUser(svc),
			},
			{
				Name: OpGetRepo, Description: "Get a repository", Scopes: []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
				},
				Handler: invokeGetRepo(svc),
			},
			{
				Name: OpListIssues, Description: "List issues", Scopes: []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "state", Type: "string", Required: false, Description: "Issue state filter (open/closed)"},
				},
				Handler: invokeListIssues(svc),
			},
			{
				Name: OpGetIssue, Description: "Get an issue", Scopes: []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "index", Type: "int64", Required: true, Description: "Issue index number"},
				},
				Handler: invokeGetIssue(svc),
			},
			{
				Name: OpGetCommitDiff, Description: "Get commit diff", Scopes: []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "commit_id", Type: "string", Required: true, Description: "Commit hash"},
				},
				Handler: invokeGetCommitDiff(svc),
			},
			{
				Name: OpGetFileContent, Description: "Get file content", Scopes: []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "commit_id", Type: "string", Required: true, Description: "Commit hash"},
					{Name: "file_path", Type: "string", Required: true, Description: "File path"},
					{Name: "line_start", Type: "int", Required: false, Description: "Starting line number"},
					{Name: "line_count", Type: "int", Required: false, Description: "Number of lines to fetch"},
				},
				Handler: invokeGetFileContent(svc),
			},
		},
	})
}

func invokeGetUser(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		user, err := svc.GetUser(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: user, Text: user.UserName}, nil
	}
}

func invokeGetRepo(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		owner, err := capability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := capability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetRepo(ctx, owner, repo)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.FullName}, nil
	}
}

func invokeListIssues(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		owner, err := capability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		q := &ListIssuesQuery{Page: capability.PageRequestFromParams(params)}
		if state, ok := capability.StringParam(params, "state"); ok {
			q.State = state
		}
		result, err := svc.ListIssues(ctx, owner, q)
		if err != nil {
			return nil, err
		}
		return listInvokeResult(OpListIssues, result), nil
	}
}

func invokeGetIssue(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		owner, err := capability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := capability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		index, err := capability.RequiredInt64(params, "index")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetIssue(ctx, owner, repo, index)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: fmt.Sprintf("#%d %s", item.Index, item.Title)}, nil
	}
}

func invokeGetCommitDiff(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		owner, err := capability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := capability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		commitID, err := capability.RequiredString(params, "commit_id")
		if err != nil {
			return nil, err
		}
		diff, err := svc.GetCommitDiff(ctx, owner, repo, commitID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: diff, Text: diff.CommitMessage}, nil
	}
}

func invokeGetFileContent(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		owner, err := capability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := capability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		commitID, err := capability.RequiredString(params, "commit_id")
		if err != nil {
			return nil, err
		}
		filePath, err := capability.RequiredString(params, "file_path")
		if err != nil {
			return nil, err
		}
		lineStart, _ := capability.IntParam(params, "line_start")
		lineCount, _ := capability.IntParam(params, "line_count")
		content, err := svc.GetFileContent(ctx, owner, repo, commitID, filePath, lineStart, lineCount)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: string(content), Text: filePath}, nil
	}
}

func listInvokeResult(operation string, result *capability.ListResult[capability.ForgeIssue]) *capability.InvokeResult {
	if result == nil {
		result = &capability.ListResult[capability.ForgeIssue]{Items: []*capability.ForgeIssue{}, Page: &capability.PageInfo{}}
	}
	return &capability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
