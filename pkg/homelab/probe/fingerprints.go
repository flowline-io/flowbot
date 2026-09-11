package probe

// ServiceFingerprint associates a capability (provider ID) and detection
// paths for known services. Matching is path-reachability only.
type ServiceFingerprint struct {
	Capability string
	Provider   string
	Paths      []string
}

// KnownServices contains fingerprints for services that the probe engine can
// identify by reachable API paths.
var KnownServices = []ServiceFingerprint{
	{
		Capability: "karakeep",
		Provider:   "karakeep",
		Paths:      []string{"/api/v1/health"},
	},
	{
		Capability: "kanboard",
		Provider:   "kanboard",
		Paths:      []string{"/jsonrpc.php"},
	},
	{
		Capability: "miniflux",
		Provider:   "miniflux",
		Paths:      []string{"/v1/healthcheck"},
	},
	{
		Capability: "finance",
		Provider:   "fireflyiii",
		Paths:      []string{"/api/v1/about"},
	},
	{
		Capability: "archive",
		Provider:   "archivebox",
		Paths:      []string{"/admin"},
	},
}
