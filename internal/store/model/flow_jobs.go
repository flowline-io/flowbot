package model

import "time"

// FlowJob records per-node execution details for a flow execution.
// It is used by IF-THEN flows for execution tracing.
type FlowJob struct {
	ID          int64      `json:"id"`
	FlowID      int64      `json:"flow_id"`
	ExecutionID string     `json:"execution_id"`
	NodeID      string     `json:"node_id"`
	NodeType    NodeType   `json:"node_type"`
	Bot         string     `json:"bot"`
	RuleID      string     `json:"rule_id"`
	Attempt     int        `json:"attempt"`
	State       JobState   `json:"state"`
	Params      JSON       `json:"params"`
	Result      JSON       `json:"result"`
	Error       string     `json:"error"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
