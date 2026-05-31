package github

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Descriptor returns the hub capability descriptor for the GitHub capability.
func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapGithub,
		Backend:     backend,
		App:         app,
		Description: "GitHub capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{
				Name:        ability.OpGithubGetUser,
				Description: "Get authenticated user",
				Scopes:      []string{auth.ScopeServiceForgeRead},
			},
			{
				Name:        ability.OpGithubGetUserByLogin,
				Description: "Get user by login",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "login", Type: "string", Required: true, Description: "GitHub username"},
				},
			},
			{
				Name:        ability.OpGithubGetRepo,
				Description: "Get a repository",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
				},
			},
			{
				Name:        ability.OpGithubListIssues,
				Description: "List issues",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
					{Name: "state", Type: "string", Required: false, Description: "Issue state (open/closed/all)"},
				},
			},
			{
				Name:        ability.OpGithubGetIssue,
				Description: "Get an issue",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "number", Type: "int64", Required: true, Description: "Issue number"},
				},
			},
			{
				Name:        ability.OpGithubGetCommitDiff,
				Description: "Get commit diff",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "commit_id", Type: "string", Required: true, Description: "Commit hash"},
				},
			},
			{
				Name:        ability.OpGithubGetFileContent,
				Description: "Get file content",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "commit_id", Type: "string", Required: true, Description: "Commit hash"},
					{Name: "file_path", Type: "string", Required: true, Description: "File path"},
					{Name: "line_start", Type: "int", Required: false, Description: "Starting line number"},
					{Name: "line_count", Type: "int", Required: false, Description: "Number of lines to fetch"},
				},
			},
			{
				Name:        ability.OpGithubListNotifications,
				Description: "List notifications",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
			},
			{
				Name:        ability.OpGithubListReleases,
				Description: "List releases",
				Scopes:      []string{auth.ScopeServiceForgeRead},
				Input: []hub.ParamDef{
					{Name: "owner", Type: "string", Required: true, Description: "Repository owner"},
					{Name: "repo", Type: "string", Required: true, Description: "Repository name"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
					{Name: "sort_by", Type: "string", Required: false, Description: "Field to sort by"},
					{Name: "sort_order", Type: "string", Required: false, Description: "Sort order (asc/desc)"},
				},
			},
		},
		Events: []hub.EventDef{
			{Name: types.EventForgePush, Description: "Fires when code is pushed"},
		},
	}
}

// RegisterService registers the GitHub capability with the hub and ability registry.
// It returns nil and logs a warning when svc is nil (provider not configured).
func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		flog.Warn("github capability: service is nil, skipping registration for %s/%s", backend, app)
		return nil
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpGithubGetUser, invoker: invokeGetUser(svc)},
		{operation: ability.OpGithubGetUserByLogin, invoker: invokeGetUserByLogin(svc)},
		{operation: ability.OpGithubGetRepo, invoker: invokeGetRepo(svc)},
		{operation: ability.OpGithubListIssues, invoker: invokeListIssues(svc)},
		{operation: ability.OpGithubGetIssue, invoker: invokeGetIssue(svc)},
		{operation: ability.OpGithubGetCommitDiff, invoker: invokeGetCommitDiff(svc)},
		{operation: ability.OpGithubGetFileContent, invoker: invokeGetFileContent(svc)},
		{operation: ability.OpGithubListNotifications, invoker: invokeListNotifications(svc)},
		{operation: ability.OpGithubListReleases, invoker: invokeListReleases(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapGithub, item.operation, item.invoker); err != nil {
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
func invokeGetUserByLogin(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		login, err := ability.RequiredString(params, "login")
		if err != nil {
			return nil, err
		}
		user, err := svc.GetUserByLogin(ctx, login)
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
		return listIssueInvokeResult(ability.OpGithubListIssues, result), nil
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
		number, err := ability.RequiredInt64(params, "number")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetIssue(ctx, owner, repo, number)
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

func invokeListNotifications(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		q := &PageQuery{Page: ability.PageRequestFromParams(params)}
		result, err := svc.ListNotifications(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Notification]{Items: []*ability.Notification{}, Page: &ability.PageInfo{}}
		}
		return &ability.InvokeResult{Operation: ability.OpGithubListNotifications, Data: result.Items, Page: result.Page}, nil
	}
}

func invokeListReleases(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		owner, err := ability.RequiredString(params, "owner")
		if err != nil {
			return nil, err
		}
		repo, err := ability.RequiredString(params, "repo")
		if err != nil {
			return nil, err
		}
		q := &PageQuery{Page: ability.PageRequestFromParams(params)}
		result, err := svc.ListReleases(ctx, owner, repo, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &ability.ListResult[ability.Release]{Items: []*ability.Release{}, Page: &ability.PageInfo{}}
		}
		return &ability.InvokeResult{Operation: ability.OpGithubListReleases, Data: result.Items, Page: result.Page}, nil
	}
}

func listIssueInvokeResult(operation string, result *ability.ListResult[ability.ForgeIssue]) *ability.InvokeResult {
	if result == nil {
		result = &ability.ListResult[ability.ForgeIssue]{Items: []*ability.ForgeIssue{}, Page: &ability.PageInfo{}}
	}
	return &ability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
