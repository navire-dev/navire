package ntfy

import (
	"encoding/base64"
	"fmt"

	"github.com/navire-dev/navire/core/internal/provider/urlprep"
	"github.com/navire-dev/navire/shared/models"
)

const AuthParam models.AuthParam = "auth"

type Provider struct{}

func (Provider) PrepareURL(endpoint models.Endpoint) (string, error) {
	if endpoint.Auth.Type == models.AuthQuery {
		if endpoint.Auth.Param != AuthParam {
			return "", fmt.Errorf("ntfy query auth must use parameter %q", AuthParam)
		}
		endpoint.Auth.Value = base64.RawStdEncoding.EncodeToString([]byte("Bearer " + endpoint.Auth.Value))
	}
	return urlprep.Prepare(endpoint.URL, endpoint.Auth)
}
