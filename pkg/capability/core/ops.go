// Package core registers hub.CapCore — internal multi-op capability for notify, clip,
// agent, HTTP, sandboxed execution, and persistent KV.
package core

const (
	OpHealth = "health"

	OpNotifySend   = "notify_send"
	OpNotifyHealth = "notify_health"

	OpClipCreate = "clip_create"
	OpClipGet    = "clip_get"
	OpClipHealth = "clip_health"

	OpAgentRun    = "agent_run"
	OpAgentHealth = "agent_health"

	OpHTTPRequest = "http_request"

	OpRunCode     = "run_code"
	OpRunTerminal = "run_terminal"

	OpKVGet    = "kv_get"
	OpKVSet    = "kv_set"
	OpKVDelete = "kv_delete"
)
