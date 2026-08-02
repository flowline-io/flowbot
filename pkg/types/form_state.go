package types

import "database/sql/driver"

// FormState is the lifecycle state of a persisted form.
type FormState int

const (
	FormStateUnknown FormState = iota
	FormStateCreated
	FormStateSubmitSuccess
	FormStateSubmitFailed
)

// Value implements driver.Valuer for database persistence.
func (j FormState) Value() (driver.Value, error) {
	return int64(j), nil
}
