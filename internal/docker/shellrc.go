// shellrc.go generates shell RC files with aliases and welcome messages.
package docker

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/peasant-labs/zone/pkg/templates"
)

// ShellRCData holds all variables consumed by zone-bashrc.tmpl.
type ShellRCData struct {
	HarnessName    string
	MountPath      string
	Aliases        map[string]string
	ShellRC        []string
	WelcomeMessage string
}

// RenderShellRC renders the zone-bashrc template with the given data.
// The generation header is prepended (the template's own static header is kept as-is).
func RenderShellRC(data ShellRCData, version string) (string, error) {
	tmpl, err := template.New("zone-bashrc").Funcs(templateFuncs()).Parse(templates.ZoneBashrcTmpl)
	if err != nil {
		return "", fmt.Errorf("parse zone-bashrc template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render zone-bashrc: %w", err)
	}
	return injectGenerationComment(buf.String(), version), nil
}
