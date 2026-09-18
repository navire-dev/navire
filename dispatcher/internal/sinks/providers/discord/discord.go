package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/shared/models"
)

func Prepare(endpoint models.Endpoint, message messageTemplate.Rendered) (contract.PreparedRequest, error) {
	if err := validateEmbed(message); err != nil {
		return contract.PreparedRequest{}, err
	}
	targetURL, err := webhookURL(endpoint.URL)
	if err != nil {
		return contract.PreparedRequest{}, err
	}
	payload, err := json.Marshal(struct {
		Embeds          []embed         `json:"embeds"`
		AllowedMentions allowedMentions `json:"allowed_mentions"`
	}{
		Embeds:          []embed{{Title: message.Title, Description: message.Body}},
		AllowedMentions: allowedMentions{Parse: []string{}},
	})
	if err != nil {
		return contract.PreparedRequest{}, fmt.Errorf("encode discord payload: %w", err)
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return contract.PreparedRequest{Provider: "discord", Method: http.MethodPost, URL: targetURL, Headers: headers, Body: payload}, nil
}

type embed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type allowedMentions struct {
	Parse []string `json:"parse"`
}

func validateEmbed(message messageTemplate.Rendered) error {
	if utf8.RuneCountInString(message.Title) > 256 {
		return fmt.Errorf("discord embed title exceeds 256 characters")
	}
	if utf8.RuneCountInString(message.Body) > 4096 {
		return fmt.Errorf("discord embed description exceeds 4096 characters")
	}
	if utf8.RuneCountInString(message.Title)+utf8.RuneCountInString(message.Body) > 6000 {
		return fmt.Errorf("discord embed exceeds 6000 characters")
	}
	return nil
}

func webhookURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse discord webhook URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("discord webhook URL must use https")
	}
	query := parsed.Query()
	query.Set("wait", "true")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
