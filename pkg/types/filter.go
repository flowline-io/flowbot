package types

import "time"

type JobFilter struct {
	EndedAt time.Time
}

type DataFilter struct {
	Prefix       *string
	CreatedStart *time.Time
	CreatedEnd   *time.Time
}
