package ntfy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/shared/models"
)

func Prepare(endpoint models.Endpoint, message messageTemplate.Rendered) (contract.PreparedRequest, error) {
	targetURL, err := validateURL(endpoint.URL)
	if err != nil {
		return contract.PreparedRequest{}, err
	}
	body := []byte(formatBody(message.Body))
	headers := make(http.Header)
	headers.Set("Content-Type", "text/markdown")
	headers.Set("Title", message.Title)
	headers.Set("Priority", priority(message.Priority))
	return contract.PreparedRequest{Provider: "ntfy", Method: http.MethodPost, URL: targetURL, Headers: headers, Body: body}, nil
}

func validateURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse ntfy URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("ntfy URL must use http or https and include a host")
	}
	return parsed.String(), nil
}

// formatBody adapts Navire's canonical Markdown to Ntfy's compact Markdown
// rendering. Gotify uses blank lines to separate Markdown paragraphs, while
// Ntfy displays those separators as an unwanted visible gap in notifications.
func formatBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	for strings.Contains(body, "\n\n") {
		body = strings.ReplaceAll(body, "\n\n", "\n")
	}
	return strings.TrimSpace(body)
}

func priority(value messageTemplate.Priority) string {
	switch value {
	case messageTemplate.PriorityLow:
		return "low"
	case messageTemplate.PriorityHigh:
		return "high"
	case messageTemplate.PriorityCritical:
		return "urgent"
	default:
		return "default"
	}
}
