// Package dozzle implements the Dozzle Docker log viewer provider (health-only MVP).
package dozzle

// VersionInfo holds Dozzle version text from /api/version.
type VersionInfo struct {
	Version string `json:"version"`
}
