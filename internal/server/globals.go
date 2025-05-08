package server

import (
	"time"

	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var globals struct {
	// Add Strict-Transport-Security to headers, the value signifies age.
	// Empty string "" turns it off
	tlsStrictMaxAge string
	// Listen for connections on this address:port and redirect them to HTTPS port.
	tlsRedirectHTTP string

	// Maximum allowed upload size.
	maxFileUploadSize int64
	// Periodicity of a garbage collector for abandoned media uploads.
	mediaGcPeriod time.Duration

	// Cron
	cronRuleset []*cron.Ruleset

	// Workflow
	taskQueue       *workflow.Queue
	manager         *workflow.Manager
	cronTaskManager *workflow.CronTaskManager
}
