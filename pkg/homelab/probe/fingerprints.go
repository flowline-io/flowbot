package probe

// FingerprintPattern describes a single detection pattern for a known service.
type FingerprintPattern struct {
	Field string // "header", "body_key", "path", "title"
	Key   string // header name / JSON key / URL path
	Value string // regex pattern to match
}

// ServiceFingerprint associates a capability type, backend provider and
// detection patterns for known services.
type ServiceFingerprint struct {
	Capability string
	Provider   string
	Patterns   []FingerprintPattern
}

// KnownServices contains fingerprints for services that the probe engine can
// identify by response patterns.
var KnownServices = []ServiceFingerprint{
	{
		Capability: "bookmark",
		Provider:   "karakeep",
		Patterns: []FingerprintPattern{
			{Field: "header", Key: "Server", Value: "LinkWarden"},
			{Field: "path", Key: "/api/v1/health", Value: ""},
		},
	},
	{
		Capability: "kanban",
		Provider:   "kanboard",
		Patterns: []FingerprintPattern{
			{Field: "title", Key: "", Value: "Kanboard"},
			{Field: "path", Key: "/jsonrpc.php", Value: ""},
		},
	},
	{
		Capability: "reader",
		Provider:   "miniflux",
		Patterns: []FingerprintPattern{
			{Field: "header", Key: "X-Auth-Token", Value: ""},
			{Field: "path", Key: "/v1/healthcheck", Value: ""},
		},
	},
	{
		Capability: "finance",
		Provider:   "fireflyiii",
		Patterns: []FingerprintPattern{
			{Field: "header", Key: "X-Firefly-III-Version", Value: ""},
			{Field: "path", Key: "/api/v1/about", Value: ""},
		},
	},
	{
		Capability: "archive",
		Provider:   "archivebox",
		Patterns: []FingerprintPattern{
			{Field: "title", Key: "", Value: "ArchiveBox"},
			{Field: "path", Key: "/admin", Value: ""},
		},
	},
}
