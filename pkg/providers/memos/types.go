// Package memos implements the Memos provider for note-taking and knowledge management.
package memos

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
)

// MaxPageSize is the maximum number of items per page for list requests.
const MaxPageSize = 100

// visibilityNames maps protobuf Visibility enum numbers to symbolic names.
// See usememos/memos proto/api/v1/memo_service.proto.
var visibilityNames = map[int64]string{
	0: "VISIBILITY_UNSPECIFIED",
	1: "PRIVATE",
	2: "PROTECTED",
	3: "PUBLIC",
}

// stateNames maps protobuf State enum numbers to symbolic names.
// See usememos/memos proto/api/v1/common.proto.
var stateNames = map[int64]string{
	0: "STATE_UNSPECIFIED",
	1: "NORMAL",
	2: "ARCHIVED",
}

// Memo represents a memo in the Memos API.
type Memo struct {
	Name        string         `json:"name,omitempty"`
	State       string         `json:"state,omitempty"`
	Creator     string         `json:"creator,omitempty"`
	CreateTime  *time.Time     `json:"createTime,omitempty"`
	UpdateTime  *time.Time     `json:"updateTime,omitempty"`
	Content     string         `json:"content,omitempty"`
	Visibility  string         `json:"visibility,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Pinned      bool           `json:"pinned,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	Relations   []MemoRelation `json:"relations,omitempty"`
	Reactions   []Reaction     `json:"reactions,omitempty"`
	Property    *MemoProperty  `json:"property,omitempty"`
	Parent      *string        `json:"parent,omitempty"`
	Snippet     string         `json:"snippet,omitempty"`
	Location    *Location      `json:"location,omitempty"`
}

// UnmarshalJSON accepts both REST (protojson string enums) and webhook
// (encoding/json numeric enums) payloads from Memos.
func (m *Memo) UnmarshalJSON(data []byte) error {
	type memoJSON Memo
	aux := &struct {
		State      *protoEnum `json:"state"`
		Visibility *protoEnum `json:"visibility"`
		*memoJSON
	}{
		memoJSON: (*memoJSON)(m),
	}
	if err := sonic.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.State != nil {
		m.State = aux.State.resolve(stateNames)
	}
	if aux.Visibility != nil {
		m.Visibility = aux.Visibility.resolve(visibilityNames)
	}
	return nil
}

// protoEnum unmarshals a protobuf enum that may be a symbolic name or a number.
type protoEnum struct {
	name string
	num  *int64
}

// UnmarshalJSON accepts a JSON string or number for a protobuf enum field.
func (e *protoEnum) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var s string
	if err := sonic.Unmarshal(data, &s); err == nil {
		e.name = s
		return nil
	}
	var n int64
	if err := sonic.Unmarshal(data, &n); err == nil {
		e.num = &n
		return nil
	}
	return fmt.Errorf("protobuf enum: expected string or number, got %s", string(data))
}

// resolve returns the symbolic enum name, mapping known numeric values when needed.
func (e *protoEnum) resolve(names map[int64]string) string {
	if e == nil {
		return ""
	}
	if e.num != nil {
		if name, ok := names[*e.num]; ok {
			return name
		}
		return strconv.FormatInt(*e.num, 10)
	}
	return e.name
}

// MemoProperty holds computed properties of a memo.
type MemoProperty struct {
	HasLink            bool   `json:"hasLink,omitempty"`
	HasTaskList        bool   `json:"hasTaskList,omitempty"`
	HasCode            bool   `json:"hasCode,omitempty"`
	HasIncompleteTasks bool   `json:"hasIncompleteTasks,omitempty"`
	Title              string `json:"title,omitempty"`
}

// Location holds geolocation data for a memo.
type Location struct {
	Placeholder string  `json:"placeholder,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// Attachment represents a file attachment on a memo.
type Attachment struct {
	Name       string     `json:"name,omitempty"`
	Type       string     `json:"type,omitempty"`
	Size       int64      `json:"size,omitempty"`
	CreateTime *time.Time `json:"createTime,omitempty"`
	URL        string     `json:"url,omitempty"`
}

// MemoRelation represents a relation between two memos.
type MemoRelation struct {
	Memo        *MemoRelationMemo `json:"memo,omitempty"`
	RelatedMemo *MemoRelationMemo `json:"relatedMemo,omitempty"`
	Type        string            `json:"type,omitempty"`
}

// MemoRelationMemo is a lightweight memo reference inside a relation.
type MemoRelationMemo struct {
	Name    string `json:"name,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// Reaction represents a reaction on a memo.
type Reaction struct {
	Name         string     `json:"name,omitempty"`
	Creator      string     `json:"creator,omitempty"`
	ContentID    string     `json:"contentId,omitempty"`
	ReactionType string     `json:"reactionType,omitempty"`
	CreateTime   *time.Time `json:"createTime,omitempty"`
}

// MemoShare represents a share link for a memo.
type MemoShare struct {
	Name       string     `json:"name,omitempty"`
	CreateTime *time.Time `json:"createTime,omitempty"`
	ExpireTime *time.Time `json:"expireTime,omitempty"`
}

// User represents a user in the Memos API.
type User struct {
	Name        string     `json:"name,omitempty"`
	Role        string     `json:"role,omitempty"`
	Username    string     `json:"username,omitempty"`
	Email       string     `json:"email,omitempty"`
	DisplayName string     `json:"displayName,omitempty"`
	AvatarURL   string     `json:"avatarUrl,omitempty"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state,omitempty"`
	CreateTime  *time.Time `json:"createTime,omitempty"`
	UpdateTime  *time.Time `json:"updateTime,omitempty"`
}

// ListMemosResponse is the paginated response from ListMemos.
type ListMemosResponse struct {
	Memos         []Memo `json:"memos,omitempty"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// ListMemosParams holds optional query parameters for listing memos.
type ListMemosParams struct {
	// PageSize is the maximum number of memos to return (default 50, max 1000).
	PageSize int32
	// PageToken is the token for the next page of results.
	PageToken string
	// State filters memos by state ("NORMAL" or "ARCHIVED").
	State string
	// OrderBy specifies sort order (e.g., "create_time desc").
	OrderBy string
	// Filter is a CEL expression to filter memos.
	Filter string
}

// CreateMemoRequest is the request body for creating a memo.
type CreateMemoRequest struct {
	Memo   Memo   `json:"memo"`
	MemoID string `json:"memoId,omitempty"`
}

// UpdateMemoRequest is the request body for updating a memo.
type UpdateMemoRequest struct {
	Memo       Memo     `json:"memo"`
	UpdateMask []string `json:"updateMask,omitempty"`
}

// LinkMetadata holds metadata for a link extracted by the server.
type LinkMetadata struct {
	URL         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
}

// WebhookPayload is the request body sent by the Memos server to configured webhook URLs.
// The server sends a JSON POST with the activity type, creator, and full memo object.
type WebhookPayload struct {
	URL          string `json:"url"`
	ActivityType string `json:"activityType"`
	Creator      string `json:"creator"`
	Memo         Memo   `json:"memo"`
}
