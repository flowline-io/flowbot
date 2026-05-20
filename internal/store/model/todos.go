package model

import (
	"time"
)

// Todo mapped from table <todos>
type Todo struct {
	ID             int64     `json:"id"`
	UID            string    `json:"uid"`
	Topic          string    `json:"topic"`
	KeyResultID    int64     `json:"key_result_id"`
	ParentID       int64     `json:"parent_id"`
	Sequence       int32     `json:"sequence"`
	Content        string    `json:"content"`
	Category       string    `json:"category"`
	Remark         string    `json:"remark"`
	Priority       int32     `json:"priority"`
	IsRemindAtTime int32     `json:"is_remind_at_time"`
	RemindAt       int64     `json:"remind_at"`
	RepeatMethod   string    `json:"repeat_method"`
	RepeatRule     string    `json:"repeat_rule"`
	RepeatEndAt    int64     `json:"repeat_end_at"`
	Complete       int32     `json:"complete"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SubTodos       []*Todo   `json:"sub_todos"`
}
