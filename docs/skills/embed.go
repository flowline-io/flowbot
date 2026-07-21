// Package skills embeds Cap-ID Agent Skills for server startup import.
package skills

import "embed"

// FS holds Cap-ID skill trees under this directory (SKILL.md, references/, examples/, …).
//
//go:embed karakeep kanboard miniflux memos trilium fireflyiii transmission nocodb devops gitea github workflow
var FS embed.FS
