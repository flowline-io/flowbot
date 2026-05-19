package model

import (
	"time"
)


// FlowNode mapped from table <flow_nodes>
type FlowNode struct {
	ID         int64     `json:"id"`
	FlowID     int64     `json:"flow_id"`
	NodeID     string    `json:"node_id"`
	Type       NodeType  `json:"type"`
	Bot        string    `json:"bot"`
	RuleID     string    `json:"rule_id"`
	Label      string    `json:"label"`
	PositionX  int       `json:"position_x"`
	PositionY  int       `json:"position_y"`
	Parameters JSON      `json:"parameters"`
	Variables  JSON      `json:"variables"`
	Conditions JSON      `json:"conditions"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
