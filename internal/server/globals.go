package server

import (
	"time"

	"github.com/flowline-io/flowbot/internal/types/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/workflow"
)

var globals struct {
	// Indicator that shutdown is in progress
	shuttingDown bool

	// Add Strict-Transport-Security to headers, the value signifies age.
	// Empty string "" turns it off
	tlsStrictMaxAge string
	// Listen for connections on this address:port and redirect them to HTTPS port.
	tlsRedirectHTTP string

	// Maximum allowed upload size.
	maxFileUploadSize int64
	// Periodicity of a garbage collector for abandoned media uploads.
	mediaGcPeriod time.Duration

	// Prioritize X-Forwarded-For header as the source of IP address of the client.
	useXForwardedFor bool

	// Cron
	cronRuleset []*cron.Ruleset

	// Workflow
	taskQueue       *workflow.Queue
	manager         *workflow.Manager
	cronTaskManager *workflow.CronTaskManager
}
