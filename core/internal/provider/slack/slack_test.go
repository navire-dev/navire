package slack

import (
	"testing"

	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

func TestPrepareURL(t *testing.T) {
	got, err := (Provider{}).PrepareURL(models.Endpoint{
		Provider: sharedproviders.Slack,
		URL:      "https://hooks.slack.com/services/T000/B000/secret",
		Auth:     models.Auth{Type: models.AuthNone},
	})
	if err != nil {
		t.Fatalf("PrepareURL() error = %v", err)
	}
	if want := "https://hooks.slack.com/services/T000/B000/secret"; got != want {
		t.Fatalf("PrepareURL() = %q, want %q", got, want)
	}
}
