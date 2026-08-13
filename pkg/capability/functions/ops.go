// Package functions registers the CapFunctions capability (named FaaS invoke/get/health).
package functions

const (
	// OpInvoke invokes a published named function version.
	OpInvoke = "invoke"
	// OpGet returns public metadata for a published function.
	OpGet = "get"
	// OpHealth reports whether the functions service is ready.
	OpHealth = "health"
)
