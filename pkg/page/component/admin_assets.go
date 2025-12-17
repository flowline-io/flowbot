package component

import _ "embed"

//go:embed assets/admin.js
var adminJS string

func AdminJS() string {
	return adminJS
}
