package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func stubCheckCtx(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	return nil
}

type stubKanbanService struct{ cfg KanbanConfig }

func (s *stubKanbanService) ListTasks(ctx context.Context, _ *KanbanTaskQuery) (*capability.ListResult[capability.Task], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.TasksErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", s.cfg.TasksErr)
	}
	items := s.cfg.Tasks
	if items == nil {
		items = []*capability.Task{}
	}
	return &capability.ListResult[capability.Task]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubKanbanService) GetTask(ctx context.Context, id int) (*capability.Task, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.TaskErr != nil {
		return nil, types.WrapError(types.ErrProvider, "get failed", s.cfg.TaskErr)
	}
	if s.cfg.Task != nil {
		return s.cfg.Task, nil
	}
	return &capability.Task{ID: id, Title: "task"}, nil
}

func (s *stubKanbanService) CreateTask(ctx context.Context, req KanbanCreateTaskRequest) (*capability.Task, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title required")
	}
	if s.cfg.CreateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create failed", s.cfg.CreateErr)
	}
	if s.cfg.CreateTask != nil {
		return s.cfg.CreateTask, nil
	}
	id := s.cfg.CreateTaskID
	if id == 0 {
		id = 1
	}
	return &capability.Task{ID: id, Title: req.Title, ProjectID: req.ProjectID}, nil
}

func (s *stubKanbanService) UpdateTask(ctx context.Context, id int, req KanbanUpdateTaskRequest) (*capability.Task, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.UpdateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "update failed", s.cfg.UpdateErr)
	}
	if s.cfg.UpdateTask != nil {
		return s.cfg.UpdateTask, nil
	}
	title := req.Title
	if title == "" {
		title = "updated"
	}
	return &capability.Task{ID: id, Title: title}, nil
}

func (s *stubKanbanService) DeleteTask(ctx context.Context, id int) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.DeleteErr != nil {
		return types.WrapError(types.ErrProvider, "delete failed", s.cfg.DeleteErr)
	}
	return nil
}

func (s *stubKanbanService) MoveTask(ctx context.Context, id int, req KanbanMoveTaskRequest) (*capability.Task, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if id <= 0 || req.ColumnID <= 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "id and column required")
	}
	if s.cfg.MoveErr != nil {
		return nil, types.WrapError(types.ErrProvider, "move failed", s.cfg.MoveErr)
	}
	if s.cfg.MoveTask != nil {
		return s.cfg.MoveTask, nil
	}
	return &capability.Task{ID: id, ColumnID: req.ColumnID}, nil
}

func (s *stubKanbanService) CompleteTask(ctx context.Context, id int) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.CloseErr != nil {
		return types.WrapError(types.ErrProvider, "close failed", s.cfg.CloseErr)
	}
	return nil
}

func (s *stubKanbanService) GetColumns(ctx context.Context, _ int) ([]map[string]any, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.ColumnsErr != nil {
		return nil, types.WrapError(types.ErrProvider, "columns failed", s.cfg.ColumnsErr)
	}
	if s.cfg.Columns != nil {
		return s.cfg.Columns, nil
	}
	return []map[string]any{}, nil
}

func (s *stubKanbanService) SearchTasks(ctx context.Context, _ *KanbanSearchQuery) (*capability.ListResult[capability.Task], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.SearchErr != nil {
		return nil, types.WrapError(types.ErrProvider, "search failed", s.cfg.SearchErr)
	}
	items := s.cfg.SearchTasks
	if items == nil {
		items = []*capability.Task{}
	}
	return &capability.ListResult[capability.Task]{Items: items, Page: &capability.PageInfo{}}, nil
}

func stubKanbanFactory(_ *testing.T, cfg KanbanConfig) KanbanService {
	return &stubKanbanService{cfg: cfg}
}

type stubMemoService struct{ cfg MemoConfig }

func (s *stubMemoService) List(ctx context.Context, q *MemoListQuery) (*capability.ListResult[capability.Memo], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", s.cfg.ListErr)
	}
	limit := 20
	if q != nil && q.Page.Limit > 0 {
		limit = q.Page.Limit
	}
	items := s.cfg.ListItems
	if items == nil {
		items = []*capability.Memo{}
	}
	return &capability.ListResult[capability.Memo]{
		Items: items,
		Page:  &capability.PageInfo{Limit: limit, HasMore: s.cfg.ListNextCursor != "", NextCursor: s.cfg.ListNextCursor},
	}, nil
}

func (s *stubMemoService) Get(ctx context.Context, name string) (*capability.Memo, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "name required")
	}
	if s.cfg.GetErr != nil {
		return nil, types.WrapError(types.ErrProvider, "get failed", s.cfg.GetErr)
	}
	if s.cfg.GetItem != nil {
		return s.cfg.GetItem, nil
	}
	return &capability.Memo{Name: name}, nil
}

func (s *stubMemoService) Create(ctx context.Context, content, visibility string) (*capability.Memo, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if content == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "content required")
	}
	if s.cfg.CreateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create failed", s.cfg.CreateErr)
	}
	if s.cfg.CreateItem != nil {
		return s.cfg.CreateItem, nil
	}
	return &capability.Memo{Content: content, Visibility: visibility}, nil
}

func (s *stubMemoService) Update(ctx context.Context, name string, _ map[string]any) (*capability.Memo, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "name required")
	}
	if s.cfg.UpdateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "update failed", s.cfg.UpdateErr)
	}
	if s.cfg.UpdateItem != nil {
		return s.cfg.UpdateItem, nil
	}
	return &capability.Memo{Name: name}, nil
}

func (s *stubMemoService) Delete(ctx context.Context, name string) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name required")
	}
	if s.cfg.DeleteErr != nil {
		return types.WrapError(types.ErrProvider, "delete failed", s.cfg.DeleteErr)
	}
	return nil
}

func (s *stubMemoService) HealthCheck(ctx context.Context) (bool, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return false, err
	}
	if s.cfg.HealthErr != nil {
		return false, types.WrapError(types.ErrProvider, "health failed", s.cfg.HealthErr)
	}
	return s.cfg.HealthOk, nil
}

func (s *stubMemoService) ListRawEvents(ctx context.Context, _ string) ([]any, string, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, "", err
	}
	if s.cfg.RawErr != nil {
		return nil, "", types.WrapError(types.ErrProvider, "raw failed", s.cfg.RawErr)
	}
	return s.cfg.RawItems, s.cfg.RawCursor, nil
}

func stubMemoFactory(_ *testing.T, cfg MemoConfig) MemoService {
	return &stubMemoService{cfg: cfg}
}

type stubNoteService struct{ cfg NoteConfig }

func (s *stubNoteService) List(ctx context.Context, _ *NoteListQuery) (*capability.ListResult[capability.Note], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.ListErr != nil {
		return nil, types.WrapError(types.ErrProvider, "list failed", s.cfg.ListErr)
	}
	items := s.cfg.ListItems
	if items == nil {
		items = []*capability.Note{}
	}
	return &capability.ListResult[capability.Note]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubNoteService) Get(ctx context.Context, id string) (*capability.Note, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.GetErr != nil {
		return nil, types.WrapError(types.ErrProvider, "get failed", s.cfg.GetErr)
	}
	if s.cfg.GetItem != nil {
		return s.cfg.GetItem, nil
	}
	return &capability.Note{ID: id}, nil
}

func (s *stubNoteService) Create(ctx context.Context, title, content, typ, parentNoteID string) (*capability.Note, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title required")
	}
	if s.cfg.CreateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create failed", s.cfg.CreateErr)
	}
	if s.cfg.CreateItem != nil {
		return s.cfg.CreateItem, nil
	}
	return &capability.Note{ID: "new", Title: title, Content: content, Type: typ, ParentNoteIDs: []string{parentNoteID}}, nil
}

func (s *stubNoteService) Update(ctx context.Context, id, title, content string) (*capability.Note, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.UpdateErr != nil {
		return nil, types.WrapError(types.ErrProvider, "update failed", s.cfg.UpdateErr)
	}
	if s.cfg.UpdateItem != nil {
		return s.cfg.UpdateItem, nil
	}
	return &capability.Note{ID: id, Title: title, Content: content}, nil
}

func (s *stubNoteService) Delete(ctx context.Context, id string) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.DeleteErr != nil {
		return types.WrapError(types.ErrProvider, "delete failed", s.cfg.DeleteErr)
	}
	return nil
}

func (s *stubNoteService) GetContent(ctx context.Context, id string) (string, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return "", err
	}
	if id == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.ContentErr != nil {
		return "", types.WrapError(types.ErrProvider, "content failed", s.cfg.ContentErr)
	}
	return s.cfg.Content, nil
}

func (s *stubNoteService) SetContent(ctx context.Context, id, _ string) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.SetContentErr != nil {
		return types.WrapError(types.ErrProvider, "set content failed", s.cfg.SetContentErr)
	}
	return nil
}

func (s *stubNoteService) Search(ctx context.Context, _ string) (*capability.ListResult[capability.Note], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.SearchErr != nil {
		return nil, types.WrapError(types.ErrProvider, "search failed", s.cfg.SearchErr)
	}
	items := s.cfg.SearchItems
	if items == nil {
		items = []*capability.Note{}
	}
	return &capability.ListResult[capability.Note]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubNoteService) GetAppInfo(ctx context.Context) (*capability.Note, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.AppInfoErr != nil {
		return nil, types.WrapError(types.ErrProvider, "app info failed", s.cfg.AppInfoErr)
	}
	if s.cfg.AppInfo != nil {
		return s.cfg.AppInfo, nil
	}
	return &capability.Note{ID: "app", Title: "notes"}, nil
}

func (s *stubNoteService) ListRawEvents(ctx context.Context, _ string) ([]any, string, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, "", err
	}
	if s.cfg.RawErr != nil {
		return nil, "", types.WrapError(types.ErrProvider, "raw failed", s.cfg.RawErr)
	}
	return s.cfg.RawItems, s.cfg.RawCursor, nil
}

func stubNoteFactory(_ *testing.T, cfg NoteConfig) NoteService {
	return &stubNoteService{cfg: cfg}
}

type stubForgeService struct{ cfg ForgeConfig }

func (s *stubForgeService) GetUser(ctx context.Context) (*capability.ForgeUser, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.UserErr != nil {
		return nil, types.WrapError(types.ErrProvider, "user failed", s.cfg.UserErr)
	}
	if s.cfg.User != nil {
		return s.cfg.User, nil
	}
	return &capability.ForgeUser{ID: 1, UserName: "forge-user"}, nil
}

func (s *stubForgeService) GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner and repo required")
	}
	if s.cfg.RepoErr != nil {
		return nil, types.WrapError(types.ErrProvider, "repo failed", s.cfg.RepoErr)
	}
	if s.cfg.Repo != nil {
		return s.cfg.Repo, nil
	}
	return &capability.ForgeRepo{Name: repo, Owner: owner, FullName: owner + "/" + repo}, nil
}

func (s *stubForgeService) ListIssues(ctx context.Context, owner string, _ *ForgeListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner required")
	}
	if s.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "issues failed", s.cfg.IssuesErr)
	}
	items := s.cfg.Issues
	if items == nil {
		items = []*capability.ForgeIssue{}
	}
	return &capability.ListResult[capability.ForgeIssue]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubForgeService) GetIssue(ctx context.Context, owner, repo string, index int64) (*capability.ForgeIssue, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || index <= 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner repo index required")
	}
	if s.cfg.IssueErr != nil {
		return nil, types.WrapError(types.ErrProvider, "issue failed", s.cfg.IssueErr)
	}
	if s.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "issue failed", s.cfg.IssuesErr)
	}
	if s.cfg.Issue != nil {
		return s.cfg.Issue, nil
	}
	if len(s.cfg.Issues) > 0 {
		return s.cfg.Issues[0], nil
	}
	return &capability.ForgeIssue{Index: index, Title: "issue"}, nil
}

func (s *stubForgeService) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || commitID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner repo commit required")
	}
	if s.cfg.DiffErr != nil {
		return nil, types.WrapError(types.ErrProvider, "diff failed", s.cfg.DiffErr)
	}
	if s.cfg.Diff != nil {
		return s.cfg.Diff, nil
	}
	return &capability.ForgeCommitDiff{CommitID: commitID}, nil
}

func (s *stubForgeService) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || commitID == "" || filePath == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "path required")
	}
	if lineStart < 0 || lineCount < 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid line range")
	}
	if s.cfg.FileContentErr != nil {
		return nil, types.WrapError(types.ErrProvider, "file failed", s.cfg.FileContentErr)
	}
	if s.cfg.FileContent != nil {
		return s.cfg.FileContent, nil
	}
	return []byte("file"), nil
}

func stubForgeFactory(_ *testing.T, cfg ForgeConfig) ForgeService {
	return &stubForgeService{cfg: cfg}
}

type stubGithubService struct{ cfg GithubConfig }

func (s *stubGithubService) GetUser(ctx context.Context) (*capability.ForgeUser, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.UserErr != nil {
		return nil, types.WrapError(types.ErrProvider, "user failed", s.cfg.UserErr)
	}
	if s.cfg.User != nil {
		return s.cfg.User, nil
	}
	return &capability.ForgeUser{ID: 1, UserName: "github-user"}, nil
}

func (s *stubGithubService) GetUserByLogin(ctx context.Context, login string) (*capability.ForgeUser, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if login == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "login required")
	}
	if s.cfg.UserByLoginErr != nil {
		return nil, types.WrapError(types.ErrProvider, "user failed", s.cfg.UserByLoginErr)
	}
	if s.cfg.UserByLogin != nil {
		return s.cfg.UserByLogin, nil
	}
	return &capability.ForgeUser{UserName: login}, nil
}

func (s *stubGithubService) GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner and repo required")
	}
	if s.cfg.RepoErr != nil {
		return nil, types.WrapError(types.ErrProvider, "repo failed", s.cfg.RepoErr)
	}
	if s.cfg.Repo != nil {
		return s.cfg.Repo, nil
	}
	return &capability.ForgeRepo{Name: repo, Owner: owner}, nil
}

func (s *stubGithubService) ListIssues(ctx context.Context, owner string, _ *GithubListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner required")
	}
	if s.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "issues failed", s.cfg.IssuesErr)
	}
	items := s.cfg.Issues
	if items == nil {
		items = []*capability.ForgeIssue{}
	}
	return &capability.ListResult[capability.ForgeIssue]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubGithubService) GetIssue(ctx context.Context, owner, repo string, number int64) (*capability.ForgeIssue, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || number <= 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner repo number required")
	}
	if s.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "issue failed", s.cfg.IssuesErr)
	}
	if len(s.cfg.Issues) > 0 {
		return s.cfg.Issues[0], nil
	}
	return &capability.ForgeIssue{Index: number, Title: "gh-issue"}, nil
}

func (s *stubGithubService) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || commitID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner repo commit required")
	}
	if s.cfg.DiffErr != nil {
		return nil, types.WrapError(types.ErrProvider, "diff failed", s.cfg.DiffErr)
	}
	if s.cfg.Diff != nil {
		return s.cfg.Diff, nil
	}
	return &capability.ForgeCommitDiff{CommitID: commitID}, nil
}

func (s *stubGithubService) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" || commitID == "" || filePath == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "path required")
	}
	if lineStart < 0 || lineCount < 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid line range")
	}
	if s.cfg.FileContentErr != nil {
		return nil, types.WrapError(types.ErrProvider, "file failed", s.cfg.FileContentErr)
	}
	if s.cfg.FileContent != nil {
		return s.cfg.FileContent, nil
	}
	return []byte("github-file"), nil
}

func (s *stubGithubService) ListNotifications(ctx context.Context, _ *GithubPageQuery) (*capability.ListResult[capability.Notification], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.NotificationsErr != nil {
		return nil, types.WrapError(types.ErrProvider, "notifications failed", s.cfg.NotificationsErr)
	}
	items := s.cfg.Notifications
	if items == nil {
		items = []*capability.Notification{}
	}
	return &capability.ListResult[capability.Notification]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubGithubService) ListReleases(ctx context.Context, owner, repo string, _ *GithubPageQuery) (*capability.ListResult[capability.Release], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if owner == "" || repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner and repo required")
	}
	if s.cfg.ReleasesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "releases failed", s.cfg.ReleasesErr)
	}
	items := s.cfg.Releases
	if items == nil {
		items = []*capability.Release{}
	}
	return &capability.ListResult[capability.Release]{Items: items, Page: &capability.PageInfo{}}, nil
}

func stubGithubFactory(_ *testing.T, cfg GithubConfig) GithubService {
	return &stubGithubService{cfg: cfg}
}

type stubReaderService struct{ cfg ReaderConfig }

func (s *stubReaderService) ListFeeds(ctx context.Context, _ *capability.ReaderFeedQuery) (*capability.ListResult[capability.Feed], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.FeedsErr != nil {
		return nil, types.WrapError(types.ErrProvider, "feeds failed", s.cfg.FeedsErr)
	}
	items := s.cfg.Feeds
	if items == nil {
		items = []*capability.Feed{}
	}
	return &capability.ListResult[capability.Feed]{Items: items, Page: &capability.PageInfo{}}, nil
}

func (s *stubReaderService) CreateFeed(ctx context.Context, feedURL string) (*capability.Feed, error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if feedURL == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "feed url required")
	}
	if s.cfg.CreateFeedErr != nil {
		return nil, types.WrapError(types.ErrProvider, "create feed failed", s.cfg.CreateFeedErr)
	}
	id := s.cfg.CreateFeedID
	if id == 0 {
		id = 1
	}
	return &capability.Feed{ID: id, FeedURL: feedURL}, nil
}

func (s *stubReaderService) ListEntries(ctx context.Context, _ *capability.ReaderEntryQuery) (*capability.ListResult[capability.Entry], error) {
	if err := stubCheckCtx(ctx); err != nil {
		return nil, err
	}
	if s.cfg.EntriesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "entries failed", s.cfg.EntriesErr)
	}
	items := s.cfg.Entries
	if items == nil {
		items = []*capability.Entry{}
	}
	page := &capability.PageInfo{}
	if s.cfg.EntriesTotal > 0 {
		total := s.cfg.EntriesTotal
		page.Total = &total
	}
	return &capability.ListResult[capability.Entry]{Items: items, Page: page}, nil
}

func (s *stubReaderService) MarkEntryRead(ctx context.Context, id int64) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.MarkReadErr != nil {
		return types.WrapError(types.ErrProvider, "mark read failed", s.cfg.MarkReadErr)
	}
	return nil
}

func (s *stubReaderService) MarkEntryUnread(ctx context.Context, id int64) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.MarkUnreadErr != nil {
		return types.WrapError(types.ErrProvider, "mark unread failed", s.cfg.MarkUnreadErr)
	}
	return nil
}

func (s *stubReaderService) StarEntry(ctx context.Context, id int64) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.StarErr != nil {
		return types.WrapError(types.ErrProvider, "star failed", s.cfg.StarErr)
	}
	return nil
}

func (s *stubReaderService) UnstarEntry(ctx context.Context, id int64) error {
	if err := stubCheckCtx(ctx); err != nil {
		return err
	}
	if id <= 0 {
		return types.Errorf(types.ErrInvalidArgument, "id required")
	}
	if s.cfg.UnstarErr != nil {
		return types.WrapError(types.ErrProvider, "unstar failed", s.cfg.UnstarErr)
	}
	return nil
}

func stubReaderFactory(_ *testing.T, cfg ReaderConfig) ReaderService {
	return &stubReaderService{cfg: cfg}
}

func TestRunConformanceSuitesWithStubs(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "kanban suite with stub", run: func(t *testing.T) { RunKanbanConformance(t, stubKanbanFactory) }},
		{name: "memo suite with stub", run: func(t *testing.T) { RunMemoConformance(t, stubMemoFactory) }},
		{name: "note suite with stub", run: func(t *testing.T) { RunNoteConformance(t, stubNoteFactory) }},
		{name: "forge suite with stub", run: func(t *testing.T) { RunForgeConformance(t, stubForgeFactory) }},
		{name: "github suite with stub", run: func(t *testing.T) { RunGithubConformance(t, stubGithubFactory) }},
		{name: "reader suite with stub", run: func(t *testing.T) { RunReaderConformance(t, stubReaderFactory) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestStubKanbanServiceDirect(t *testing.T) {
	tests := []struct {
		name string
		call func(t *testing.T, svc KanbanService)
	}{
		{
			name: "create task returns configured id",
			call: func(t *testing.T, svc KanbanService) {
				item, err := svc.CreateTask(t.Context(), KanbanCreateTaskRequest{Title: "X", ProjectID: 1})
				require.NoError(t, err)
				assert.Equal(t, 1, item.ID)
			},
		},
		{
			name: "get task rejects invalid id",
			call: func(t *testing.T, svc KanbanService) {
				_, err := svc.GetTask(t.Context(), 0)
				RequireInvalidArgError(t, err)
			},
		},
		{
			name: "search returns empty by default",
			call: func(t *testing.T, svc KanbanService) {
				result, err := svc.SearchTasks(t.Context(), &KanbanSearchQuery{})
				require.NoError(t, err)
				assert.Empty(t, result.Items)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call(t, stubKanbanFactory(t, KanbanConfig{}))
		})
	}
}
