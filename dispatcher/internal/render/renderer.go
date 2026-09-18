// Package render executes the already-selected message content from an
// execution plan. It does not load or understand Navire template files.
package render

import (
	"bytes"
	"fmt"
	texttemplate "text/template"

	"github.com/navire-dev/navire/shared/models"
)

type (
	Rendered = models.Rendered
	Priority = models.Priority
)

const (
	PriorityLow      = models.PriorityLow
	PriorityNormal   = models.PriorityNormal
	PriorityHigh     = models.PriorityHigh
	PriorityCritical = models.PriorityCritical
)

func Render(title, body string, data map[string]any) (Rendered, error) {
	renderedTitle, err := execute("title", title, data)
	if err != nil {
		return Rendered{}, err
	}
	renderedBody, err := execute("body", body, data)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{Title: renderedTitle, Body: renderedBody}, nil
}

func execute(name, source string, data map[string]any) (string, error) {
	tmpl, err := texttemplate.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("compile %s: %w", name, err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return output.String(), nil
}
