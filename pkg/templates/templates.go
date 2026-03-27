// Package templates provides embedded Dockerfile and script templates for zone containers.
package templates

import "embed"

//go:embed *.tmpl
var FS embed.FS
