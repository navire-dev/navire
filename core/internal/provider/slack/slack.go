package slack

import (
	"github.com/navire-dev/navire/core/internal/provider/urlprep"
	"github.com/navire-dev/navire/shared/models"
)

type Provider struct{}

func (Provider) PrepareURL(endpoint models.Endpoint) (string, error) {
	return urlprep.Prepare(endpoint.URL, endpoint.Auth)
}
