package config

import "time"

// Normalize applies defaults and builds internal views from the public YAML surface.
// Call after unmarshaling and before Validate / opening subsystems.
func (t *Type) Normalize() {
	t.normalizePostgres()
	t.normalizeMedia()
	t.normalizeChatAgentMedia()
	t.normalizePII()
}

const (
	// DefaultPIITimeout is applied when pii.timeout is zero.
	DefaultPIITimeout = 15 * time.Second
	// DefaultPIIScoreThreshold is applied when pii.score_threshold is zero.
	DefaultPIIScoreThreshold = 0.5
	// DefaultPIISessionTTL is applied when pii.session_ttl is zero.
	DefaultPIISessionTTL = 24 * time.Hour
)

// normalizePII fills zero PII tunables with built-in defaults.
func (t *Type) normalizePII() {
	if t.PII.Timeout <= 0 {
		t.PII.Timeout = DefaultPIITimeout
	}
	if t.PII.ScoreThreshold <= 0 {
		t.PII.ScoreThreshold = DefaultPIIScoreThreshold
	}
	if t.PII.SessionTTL <= 0 {
		t.PII.SessionTTL = DefaultPIISessionTTL
	}
}

// normalizePostgres maps PostgresConfig into the internal StoreType view used by store.Open.
func (t *Type) normalizePostgres() {
	pg := t.Postgres
	adapter := map[string]any{
		"dsn": pg.DSN,
	}
	if pg.MaxOpenConns != 0 {
		adapter["max_open_conns"] = pg.MaxOpenConns
	}
	if pg.MaxIdleConns != 0 {
		adapter["max_idle_conns"] = pg.MaxIdleConns
	}
	if pg.ConnMaxLifetime != 0 {
		adapter["conn_max_lifetime"] = pg.ConnMaxLifetime
	}
	if pg.ConnMaxIdleTime != 0 {
		adapter["conn_max_idle_time"] = pg.ConnMaxIdleTime
	}
	if pg.SQLTimeout != 0 {
		adapter["sql_timeout"] = pg.SQLTimeout
	}
	if pg.HealthCheckInterval != 0 {
		adapter["pool_health_check_interval"] = pg.HealthCheckInterval
	}
	if pg.HealthCheckTimeout != 0 {
		adapter["pool_health_check_timeout"] = pg.HealthCheckTimeout
	}

	t.Store = StoreType{
		MaxResults: pg.MaxResults,
		UseAdapter: "postgres",
		Adapters: map[string]any{
			"postgres": adapter,
		},
	}
}

// normalizeMedia fills zero media tunables with built-in defaults.
func (t *Type) normalizeMedia() {
	if t.Media == nil {
		return
	}
	if t.Media.MaxFileUploadSize <= 0 {
		t.Media.MaxFileUploadSize = defaultMediaMaxSize
	}
	if t.Media.GcPeriod <= 0 {
		t.Media.GcPeriod = defaultMediaGcPeriod
	}
	if t.Media.GcBlockSize <= 0 {
		t.Media.GcBlockSize = defaultMediaGcBlockSize
	}
}

const defaultChatAgentSignedURLTTL = time.Hour

// normalizeChatAgentMedia fills multimodal media defaults.
func (t *Type) normalizeChatAgentMedia() {
	if t.ChatAgent.Media.SignedURLTTL <= 0 {
		t.ChatAgent.Media.SignedURLTTL = defaultChatAgentSignedURLTTL
	}
}
