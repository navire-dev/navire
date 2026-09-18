package urlprep

import (
	"testing"

	"github.com/navire-dev/navire/shared/models"
)

func TestPrepareAppendsPathSegmentAuth(t *testing.T) {
	got, err := Prepare("https://discord.com/api/webhooks/123/", models.Auth{Type: models.AuthPathSegment, Value: "webhook-token"})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got != "https://discord.com/api/webhooks/123/webhook-token" {
		t.Fatalf("Prepare() = %q", got)
	}
}

func TestPrepareRejectsReservedPathCharacters(t *testing.T) {
	_, err := Prepare("https://discord.com/api/webhooks/123", models.Auth{Type: models.AuthPathSegment, Value: "token/with/slash"})
	if err == nil {
		t.Fatal("Prepare() accepted reserved path characters")
	}
}
