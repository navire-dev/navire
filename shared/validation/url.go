package validation

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// ValidateHTTPURL validates an endpoint URL before it enters the runtime
// configuration projection or an execution plan.
func ValidateHTTPURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("url is required")
	}
	if strings.IndexFunc(rawURL, unicode.IsSpace) >= 0 {
		return fmt.Errorf("url must not contain whitespace")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("url host is required")
	}
	if parsed.User != nil {
		return fmt.Errorf("url userinfo is not allowed")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("url fragment is not allowed")
	}

	return nil
}
