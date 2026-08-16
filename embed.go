// Package llmwikiokf carries the embedded project template used by `llmwiki init`.
package llmwikiokf

import "embed"

// Template holds the scaffold written out by `llmwiki init`: llmwiki.yaml and the
// wiki/ OKF bundle (seed pages, index.md, log.md, and the static viewer).
//
//go:embed all:template
var Template embed.FS
