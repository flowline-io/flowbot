package model

import (
	"time"
)

// FlowEdge mapped from table <flow_edges>
type FlowEdge struct {
	ID         int64     `json:"id"`
	FlowID     int64     `json:"flow_id"`
	EdgeID     string    `json:"edge_id"`
	SourceNode string    `json:"source_node"`
	TargetNode string    `json:"target_node"`
	SourcePort string    `json:"source_port"`
	TargetPort string    `json:"target_port"`
	Label      string    `json:"label"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
