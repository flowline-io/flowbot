// Package component provides reusable go-app UI components.
package component

import _ "embed"

//go:embed assets/admin.js
var adminJS string // skipcq: SCC-compile

func AdminJS() string {
	return adminJS
}
