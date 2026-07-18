package clip

const (
	// OpCreate creates a shareable markdown clip and returns its public URL.
	OpCreate = "create"
	// OpGet loads a clip by slug.
	OpGet = "get"
	// OpHealth reports whether the clip capability is registered and ready.
	OpHealth = "health"
)
