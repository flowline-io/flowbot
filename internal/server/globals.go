package server

import (
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/workflow/manager"
	"github.com/sysatom/flowbot/internal/workflow/scheduler"
	"github.com/sysatom/flowbot/internal/workflow/worker"
	"github.com/sysatom/flowbot/pkg/channels/crawler"
	"time"
)

var globals struct {
	// Topics cache and processing.
	hub *Hub
	// Indicator that shutdown is in progress
	shuttingDown bool
	// Sessions cache.
	sessionStore *SessionStore
	// Runtime statistics communication channel.
	statsUpdate chan *varUpdate

	// Add Strict-Transport-Security to headers, the value signifies age.
	// Empty string "" turns it off
	tlsStrictMaxAge string
	// Listen for connections on this address:port and redirect them to HTTPS port.
	tlsRedirectHTTP string
	// Maximum message size allowed from peer.
	maxMessageSize int64
	// Maximum number of group topic subscribers.
	maxSubscriberCount int
	// Maximum number of indexable tags.
	maxTagCount int
	// If true, ordinary users cannot delete their accounts.
	permanentAccounts bool

	// Maximum allowed upload size.
	maxFileUploadSize int64
	// Periodicity of a garbage collector for abandoned media uploads.
	mediaGcPeriod time.Duration

	// Prioritize X-Forwarded-For header as the source of IP address of the client.
	useXForwardedFor bool

	// Country code to assign to sessions by default.
	defaultCountryCode string

	// Time before the call is dropped if not answered.
	callEstablishmentTimeout int

	// Websocket per-message compression negotiation is enabled.
	wsCompression bool

	// URL of the main endpoint.
	servingAt string

	// Extra vars
	crawler     *crawler.Crawler
	cronRuleset []*cron.Ruleset
	manager     *manager.Manager
	scheduler   *scheduler.Scheduler
	worker      *worker.Worker
}
