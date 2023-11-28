package server

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/channels/crawler"
	"time"
)

var globals struct {
	// Indicator that shutdown is in progress
	shuttingDown bool
	// Sessions cache.
	sessionStore *SessionStore

	// Add Strict-Transport-Security to headers, the value signifies age.
	// Empty string "" turns it off
	tlsStrictMaxAge string
	// Listen for connections on this address:port and redirect them to HTTPS port.
	tlsRedirectHTTP string
	// Maximum message size allowed from peer.
	maxMessageSize int64

	// Maximum allowed upload size.
	maxFileUploadSize int64
	// Periodicity of a garbage collector for abandoned media uploads.
	mediaGcPeriod time.Duration

	// Prioritize X-Forwarded-For header as the source of IP address of the client.
	useXForwardedFor bool

	// Time before the call is dropped if not answered.
	callEstablishmentTimeout int

	// Websocket per-message compression negotiation is enabled.
	wsCompression bool

	// URL of the main endpoint.
	servingAt string

	// Crawler
	crawler *crawler.Crawler

	// Cron
	cronRuleset []*cron.Ruleset

	// Workflow
	taskQueue       *workflow.Queue
	manager         *workflow.Manager
	scheduler       *workflow.Scheduler
	cronTaskManager *workflow.CronTaskManager
}
