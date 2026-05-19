package model

import (
	"time"
)


// WorkflowScript mapped from table <workflow_script>
type WorkflowScript struct {
	ID         int64              `json:"id"`
	WorkflowID int64              `json:"workflow_id"`
	Lang       WorkflowScriptLang `json:"lang"`
	Code       string             `json:"code"`
	Version    int32              `json:"version"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}
