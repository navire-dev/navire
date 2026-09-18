package gotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/shared/models"
)

func Prepare(endpoint models.Endpoint, message messageTemplate.Rendered) (contract.PreparedRequest, error) {
	targetURL, headers, err := requestTarget(endpoint)
	if err != nil {
		return contract.PreparedRequest{}, err
	}
	payload, err := json.Marshal(struct {
		Title    string                    `json:"title"`
		Message  string                    `json:"message"`
		Priority int                       `json:"priority"`
		Extras   map[string]map[string]any `json:"extras"`
	}{
		Title: message.Title, Message: formatBody(message.Body), Priority: gotifyPriority(message.Priority),
		Extras: map[string]map[string]any{"client::display": {"contentType": "text/markdown"}},
	})
	if err != nil {
		return contract.PreparedRequest{}, fmt.Errorf("encode gotify payload: %w", err)
	}
	requestHeaders := make(http.Header)
	requestHeaders.Set("Content-Type", "application/json")
	for name, value := range headers {
		requestHeaders.Set(name, value)
	}
	return contract.PreparedRequest{Provider: "gotify", Method: http.MethodPost, URL: targetURL, Headers: requestHeaders, Body: payload}, nil
}

func formatBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(strings.TrimSpace(body), "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n\n")
}

func gotifyPriority(priority messageTemplate.Priority) int {
	switch priority {
	case messageTemplate.PriorityLow:
		return 1
	case messageTemplate.PriorityHigh:
		return 8
	case messageTemplate.PriorityCritical:
		return 10
	default:
		return 5
	}
}

func requestTarget(endpoint models.Endpoint) (string, map[string]string, error) {
	parsed, err := url.Parse(endpoint.URL)
	if err != nil {
		return "", nil, fmt.Errorf("parse gotify endpoint URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", nil, fmt.Errorf("gotify endpoint URL must use http or https")
	}
	headers := make(map[string]string)
	switch endpoint.Auth.Type {
	case models.AuthQuery:
		query := parsed.Query()
		query.Set(string(endpoint.Auth.Param), endpoint.Auth.Value)
		parsed.RawQuery = query.Encode()
	case models.AuthBearer:
		headers["Authorization"] = "Bearer " + endpoint.Auth.Value
	case models.AuthHeader:
		headers[string(endpoint.Auth.Param)] = endpoint.Auth.Value
	case models.AuthNone:
	default:
		return "", nil, fmt.Errorf("unsupported gotify auth type %q", endpoint.Auth.Type)
	}
	return parsed.String(), headers, nil
}
