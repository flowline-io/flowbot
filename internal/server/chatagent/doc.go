// Package chatagent orchestrates the Flowbot chat assistant: session lifecycle,
// harness pooling, SSE streaming, tool confirmations, permissions, and scheduled tasks.
//
// HTTP routes are registered in internal/server (chatagent_http*.go); this package
// holds the service layer consumed by REST handlers and platform chat sinks.
//
// Maintainer guide: internal/server/chatagent/AGENTS.md
package chatagent
