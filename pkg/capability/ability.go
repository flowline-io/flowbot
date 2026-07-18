// Package capability provides the capability invocation framework.
package capability

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/hub"
)

// Bookmark represents a saved web bookmark with metadata and organisation flags.
type Bookmark struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Title      string    `json:"title,omitzero"`
	Summary    string    `json:"summary,omitzero"`
	Tags       []string  `json:"tags,omitzero"`
	Archived   bool      `json:"archived"`
	Favourited bool      `json:"favourited"`
	CreatedAt  time.Time `json:"created_at"`
}

// ArchiveItem represents a saved web page or document in an archive store.
type ArchiveItem struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title,omitzero"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Feed represents an RSS/Atom feed subscription in a reader service.
type Feed struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	FeedURL  string `json:"feed_url"`
	SiteURL  string `json:"site_url,omitzero"`
	Category string `json:"category,omitzero"`
}

// Entry represents a single article or post within an RSS/Atom feed.
type Entry struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content,omitzero"`
	Status      string    `json:"status"`
	Starred     bool      `json:"starred"`
	PublishedAt time.Time `json:"published_at"`
	FeedTitle   string    `json:"feed_title,omitzero"`
}

// Task represents a work item on a kanban board.
type Task struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitzero"`
	ProjectID   int      `json:"project_id"`
	ColumnID    int      `json:"column_id"`
	Tags        []string `json:"tags,omitzero"`
	Reference   string   `json:"reference,omitzero"`
}

// Host represents a discovered homelab service instance.
type Host struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitzero"`
	Status  string `json:"status"`
}

// ForgeUser represents a forge user account.
type ForgeUser struct {
	ID        int64  `json:"id"`
	UserName  string `json:"username"`
	Email     string `json:"email,omitzero"`
	AvatarURL string `json:"avatar_url,omitzero"`
}

// ForgeRepo represents a repository on a software forge.
type ForgeRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description,omitzero"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url,omitzero"`
	Owner       string `json:"owner"`
}

// ForgeIssue represents an issue on a software forge.
type ForgeIssue struct {
	ID      int64  `json:"id"`
	Index   int64  `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body,omitzero"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Author  string `json:"author,omitzero"`
}

// ForgeCommitDiff represents a commit diff on a software forge.
type ForgeCommitDiff struct {
	CommitID      string   `json:"commit_id"`
	CommitMessage string   `json:"commit_message"`
	Files         []string `json:"files"`
	DiffContent   string   `json:"diff_content"`
}

// Notification represents a GitHub notification.
type Notification struct {
	ID         string    `json:"id"`
	Reason     string    `json:"reason,omitzero"`
	Unread     bool      `json:"unread"`
	Subject    string    `json:"subject,omitzero"`
	RepoName   string    `json:"repo_name,omitzero"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastReadAt time.Time `json:"last_read_at,omitzero"`
}

// Release represents a GitHub repository release.
type Release struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name,omitzero"`
	Body        string    `json:"body,omitzero"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	HTMLURL     string    `json:"html_url,omitzero"`
	PublishedAt time.Time `json:"published_at,omitzero"`
}

// Memo represents a memo from a note-taking system such as Memos.
type Memo struct {
	Name       string    `json:"name"`
	State      string    `json:"state,omitzero"`
	Content    string    `json:"content,omitzero"`
	Visibility string    `json:"visibility,omitzero"`
	Tags       []string  `json:"tags,omitzero"`
	Pinned     bool      `json:"pinned"`
	Creator    string    `json:"creator,omitzero"`
	Snippet    string    `json:"snippet,omitzero"`
	CreateTime time.Time `json:"create_time,omitzero"`
	UpdateTime time.Time `json:"update_time,omitzero"`
}

// Note represents a note from a note-taking system such as Trilium.
type Note struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Type            string   `json:"type,omitzero"`
	Content         string   `json:"content,omitzero"`
	ParentNoteIDs   []string `json:"parent_note_ids,omitzero"`
	ChildNoteIDs    []string `json:"child_note_ids,omitzero"`
	IsProtected     bool     `json:"is_protected"`
	DateCreated     string   `json:"date_created,omitzero"`
	DateModified    string   `json:"date_modified,omitzero"`
	UtcDateCreated  string   `json:"utc_date_created,omitzero"`
	UtcDateModified string   `json:"utc_date_modified,omitzero"`
}

// Transaction represents a finance transaction from Firefly III.
type Transaction struct {
	ID              string `json:"id"`
	Type            string `json:"type,omitzero"`
	Amount          string `json:"amount,omitzero"`
	Description     string `json:"description,omitzero"`
	Date            string `json:"date,omitzero"`
	CurrencyCode    string `json:"currency_code,omitzero"`
	SourceName      string `json:"source_name,omitzero"`
	DestinationName string `json:"destination_name,omitzero"`
	CategoryName    string `json:"category_name,omitzero"`
	Notes           string `json:"notes,omitzero"`
}

// FinanceAbout holds Firefly III instance metadata.
type FinanceAbout struct {
	Version    string `json:"version"`
	APIVersion string `json:"api_version,omitzero"`
	PHPVersion string `json:"php_version,omitzero"`
	OS         string `json:"os,omitzero"`
	Driver     string `json:"driver,omitzero"`
}

// FinanceUser holds the authenticated Firefly III user.
type FinanceUser struct {
	ID    string `json:"id"`
	Email string `json:"email,omitzero"`
	Role  string `json:"role,omitzero"`
}

// Torrent represents a BitTorrent download managed by Transmission.
type Torrent struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name,omitzero"`
	Status       string  `json:"status,omitzero"`
	PercentDone  float64 `json:"percent_done,omitzero"`
	RateDownload int64   `json:"rate_download,omitzero"`
	RateUpload   int64   `json:"rate_upload,omitzero"`
	DownloadDir  string  `json:"download_dir,omitzero"`
	HashString   string  `json:"hash_string,omitzero"`
	ErrorString  string  `json:"error_string,omitzero"`
}

type InvokeResult struct {
	Capability hub.CapabilityType `json:"capability"`
	Operation  string             `json:"operation"`
	Data       any                `json:"data,omitzero"`
	Page       *PageInfo          `json:"page,omitzero"`
	Text       string             `json:"text,omitzero"`
	Meta       map[string]any     `json:"meta,omitzero"`
	Events     []EventRef         `json:"events,omitzero"`
	Resource   *ResourceMeta      `json:"_resource,omitempty"`
}

type EventRef struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	EntityID  string `json:"entity_id,omitzero"`
}

// ResourceMeta identifies a resource created by a capability mutation operation.
type ResourceMeta struct {
	EventID  string `json:"event_id"`
	EntityID string `json:"entity_id"`
	App      string `json:"app"`
}
