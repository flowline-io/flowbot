// Package cronutil validates cron expressions shared by pipeline and chat scheduler.
package cronutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/flc1125/go-cron/v4"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ValidateExpr parses a cron expression using the same field set as pipeline triggers.
func ValidateExpr(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return fmt.Errorf("empty cron expression")
	}
	if _, err := cronParser.Parse(spec); err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", spec, err)
	}
	return nil
}

// NextRun returns the next fire time after from for a validated cron expression.
func NextRun(spec string, from time.Time) (time.Time, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return time.Time{}, fmt.Errorf("empty cron expression")
	}
	schedule, err := cronParser.Parse(spec)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", spec, err)
	}
	next := schedule.Next(from.UTC())
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("cron expression %q has no future runs", spec)
	}
	return next.UTC(), nil
}
