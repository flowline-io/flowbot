// Package webassets embeds the public/ static assets directory so that
// the web module can serve CSS, JS, and other frontend files from the
// compiled binary without depending on the process working directory.
package webassets

import (
	"embed"
	"io/fs"
)

// FS is the embedded filesystem rooted at the public/ directory.
// Use it with fiber's static middleware Config.FS field.
//
//go:embed public/*
var FS embed.FS

// SubFS is the embedded filesystem with the "public" prefix stripped,
// suitable for use with fiber's static middleware Config.FS field.
var SubFS fs.FS

func init() {
	sub, err := fs.Sub(FS, "public")
	if err != nil {
		panic("webassets: failed to resolve embedded public/ directory: " + err.Error())
	}
	SubFS = sub
}
