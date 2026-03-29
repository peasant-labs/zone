// entrypoint.go generates container entrypoint scripts from templates.
package docker

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/peasant-labs/zone/pkg/templates"
)

// EntrypointData holds all variables consumed by entrypoint.sh.tmpl.
type EntrypointData struct {
	MountPath          string
	ForwardGitConfig   bool
	GitUserName        string
	GitUserEmail       string
	ConfigCopyCommands []string
	Shell              string
	EntrypointCommand  string
}

// RenderEntrypoint renders the entrypoint.sh template with the given data.
// The generation header is prepended (no # syntax= directive in shell scripts).
func RenderEntrypoint(data EntrypointData, version string) (string, error) {
	tmpl, err := template.New("entrypoint").Funcs(templateFuncs()).Parse(templates.EntrypointTmpl)
	if err != nil {
		return "", fmt.Errorf("parse entrypoint template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render entrypoint: %w", err)
	}
	return injectGenerationComment(buf.String(), version), nil
}
