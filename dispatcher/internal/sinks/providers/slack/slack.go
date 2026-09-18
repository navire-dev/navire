package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/shared/models"
)

const (
	maxResponseBytes = 64 << 10
	maxHeaderRunes   = 150
	maxSectionRunes  = 3000
)

var markdownLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)

type payload struct {
	Text   string  `json:"text"`
	Blocks []block `json:"blocks,omitempty"`
}

type block struct {
	Type string `json:"type"`
	Text *text  `json:"text,omitempty"`
}

type text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func Prepare(endpoint models.Endpoint, message messageTemplate.Rendered) (contract.PreparedRequest, error) {
	targetURL, err := validateURL(endpoint.URL)
	if err != nil {
		return contract.PreparedRequest{}, err
	}
	if err := validateMessage(message); err != nil {
		return contract.PreparedRequest{}, err
	}

	bodyText := slackMarkdown(message.Body)
	payloadData := payload{
		Text: fallbackText(message.Title, bodyText),
		Blocks: []block{
			{Type: "header", Text: &text{Type: "plain_text", Text: plainTitle(message.Title)}},
			{Type: "section", Text: &text{Type: "mrkdwn", Text: bodyText}},
		},
	}
	body, err := json.Marshal(payloadData)
	if err != nil {
		return contract.PreparedRequest{}, fmt.Errorf("encode slack payload: %w", err)
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return contract.PreparedRequest{Provider: "slack", Method: http.MethodPost, URL: targetURL, Headers: headers, Body: body}, nil
}

func validateURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse slack webhook URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() != "hooks.slack.com" && parsed.Hostname() != "hooks.slack-gov.com" {
		return "", fmt.Errorf("slack webhook URL must use an HTTPS Slack webhook host")
	}
	if !strings.HasPrefix(parsed.Path, "/services/") {
		return "", fmt.Errorf("slack webhook URL must use the /services/ path")
	}
	return parsed.String(), nil
}

func validateMessage(message messageTemplate.Rendered) error {
	if utf8.RuneCountInString(strings.TrimSpace(message.Title)) > maxHeaderRunes {
		return fmt.Errorf("slack header exceeds %d characters", maxHeaderRunes)
	}
	if utf8.RuneCountInString(slackMarkdown(message.Body)) > maxSectionRunes {
		return fmt.Errorf("slack section exceeds %d characters", maxSectionRunes)
	}
	return nil
}

func fallbackText(title, body string) string {
	title = plainTitle(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return body
	}
	if body == "" {
		return title
	}
	return title + "\n" + body
}

func slackMarkdown(body string) string {
	body = markdownLink.ReplaceAllString(body, "<$2|$1>")
	body = strings.ReplaceAll(body, "**", "*")
	return strings.TrimSpace(body)
}

func plainTitle(title string) string {
	return strings.TrimSpace(strings.NewReplacer("**", "", "__", "", "`", "").Replace(title))
}
