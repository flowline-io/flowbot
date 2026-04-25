package client

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
)

// UserClient provides access to the user API.
type UserClient struct {
	c *Client
}

// Dashboard returns dashboard data.
func (u *UserClient) Dashboard(ctx context.Context) (types.KV, error) {
	var result types.KV
	err := u.c.Get(ctx, "/service/user/dashboard", &result)
	return result, err
}

// MetricsResult contains system metrics data.
type MetricsResult struct {
	BotTotalStats             int64 `json:"bot_total_stats,omitempty"`
	BookmarkTotalStats        int64 `json:"bookmark_total_stats,omitempty"`
	TorrentDownloadTotalStats int64 `json:"torrent_download_total_stats,omitempty"`
	GiteaIssueTotalStats      int64 `json:"gitea_issue_total_stats,omitempty"`
	ReaderUnreadTotalStats    int64 `json:"reader_unread_total_stats,omitempty"`
	KanbanTaskTotalStats      int64 `json:"kanban_task_total_stats,omitempty"`
	MonitorUpTotalStats       int64 `json:"monitor_up_total_stats,omitempty"`
	MonitorDownTotalStats     int64 `json:"monitor_down_total_stats,omitempty"`
	DockerContainerTotalStats int64 `json:"docker_container_total_stats,omitempty"`
}

// Metrics returns system metrics.
func (u *UserClient) Metrics(ctx context.Context) (*MetricsResult, error) {
	var result MetricsResult
	err := u.c.Get(ctx, "/service/user/metrics", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// KanbanList returns the user's kanban task list.
func (u *UserClient) KanbanList(ctx context.Context) ([]kanboard.Task, error) {
	var result []kanboard.Task
	err := u.c.Get(ctx, "/service/user/kanban", &result)
	return result, err
}

// BookmarkList returns the user's bookmark list.
func (u *UserClient) BookmarkList(ctx context.Context) ([]karakeep.Bookmark, error) {
	var result []karakeep.Bookmark
	err := u.c.Get(ctx, "/service/user/bookmark", &result)
	return result, err
}
