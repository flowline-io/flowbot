// Package skills embeds Cap-ID Agent Skills for server startup import.
package skills

import "embed"

// FS holds Cap-ID skill trees under this directory (SKILL.md + references/).
//
//go:embed karakeep kanboard miniflux memos trilium fireflyiii transmission nocodb gitea github
var FS embed.FS
