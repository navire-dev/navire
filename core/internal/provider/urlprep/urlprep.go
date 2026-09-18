package urlprep

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/navire-dev/navire/shared/models"
	sharedvalidation "github.com/navire-dev/navire/shared/validation"
)

func Prepare(rawURL string, auth models.Auth) (string, error) {
	if err := sharedvalidation.ValidateHTTPURL(rawURL); err != nil {
		return "", err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse endpoint URL: %w", err)
	}
	switch auth.Type {
	case models.AuthNone:
	case models.AuthQuery:
		query := parsed.Query()
		query.Set(string(auth.Param), auth.Value)
		parsed.RawQuery = query.Encode()
	case models.AuthPathSegment:
		if strings.ContainsAny(auth.Value, "/?#") {
			return "", fmt.Errorf("path segment contains reserved URL characters")
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + auth.Value
	default:
		return "", fmt.Errorf("auth type %q cannot be represented by a URL", auth.Type)
	}
	return parsed.String(), nil
}
