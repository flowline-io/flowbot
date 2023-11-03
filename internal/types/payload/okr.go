package payload

import "github.com/flowline-io/flowbot/internal/store/model"

type Objective struct {
	Item     model.Objective `json:"item"`
	Progress int32           `json:"progress"`
}
