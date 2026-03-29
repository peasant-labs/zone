// Package templates provides embedded Dockerfile and script templates for zone containers.
package templates

import _ "embed"

//go:embed Dockerfile.tmpl
var DockerfileTmpl string

//go:embed entrypoint.sh.tmpl
var EntrypointTmpl string

//go:embed zone-bashrc.tmpl
var ZoneBashrcTmpl string
