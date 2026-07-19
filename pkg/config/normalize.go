package config

// Normalize applies defaults and builds internal views from the public YAML surface.
// Call after unmarshaling and before Validate / opening subsystems.
func (t *Type) Normalize() {
	t.normalizePostgres()
	t.normalizeMedia()
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
