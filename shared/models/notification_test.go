package models

import "testing"

func TestEndpointValidateRejectsUnsafeURL(t *testing.T) {
	endpoint := Endpoint{
		Key:  "operations",
		URL:  "https://user:password@example.com/notify",
		Auth: Auth{Type: AuthNone},
	}

	if err := endpoint.Validate(); err == nil {
		t.Fatal("Validate() accepted endpoint URL with userinfo")
	}
}
