package types

import "database/sql/driver"

// InstructState is the lifecycle state of an instruct message.
type InstructState int

const (
	InstructStateUnknown InstructState = iota
	InstructCreate
	InstructDone
	InstructCancel
)

// Value implements driver.Valuer for database persistence.
func (j InstructState) Value() (driver.Value, error) {
	return int64(j), nil
}

// InstructObject identifies the target of an instruct message.
type InstructObject string

const (
	InstructObjectAgent InstructObject = "agent"
)

// Value implements driver.Valuer for database persistence.
func (j InstructObject) Value() (driver.Value, error) {
	return string(j), nil
}

// InstructPriority is the scheduling priority of an instruct message.
type InstructPriority int

const (
	InstructPriorityHigh    InstructPriority = 3
	InstructPriorityDefault InstructPriority = 2
	InstructPriorityLow     InstructPriority = 1
)

// Value implements driver.Valuer for database persistence.
func (j InstructPriority) Value() (driver.Value, error) {
	return int64(j), nil
}
