package ntfy

import (
	"testing"

	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

func TestProviderPreparesTokenURL(t *testing.T) {
	got, err := (Provider{}).PrepareURL(models.Endpoint{
		Provider: sharedproviders.Ntfy,
		URL:      "https://ntfy.example.com/mysecrets",
		Auth:     models.Auth{Type: models.AuthQuery, Param: AuthParam, Value: "token"},
	})
	if err != nil {
		t.Fatalf("PrepareURL() error = %v", err)
	}
	if got != "https://ntfy.example.com/mysecrets?auth=QmVhcmVyIHRva2Vu" {
		t.Fatalf("PrepareURL() = %q", got)
	}
}

func TestProviderRejectsUnsupportedQueryParameter(t *testing.T) {
	_, err := (Provider{}).PrepareURL(models.Endpoint{
		Provider: sharedproviders.Ntfy,
		URL:      "https://ntfy.example.com/mysecrets",
		Auth:     models.Auth{Type: models.AuthQuery, Param: "token", Value: "token"},
	})
	if err == nil {
		t.Fatal("PrepareURL() accepted an unsupported query parameter")
	}
}
