package validation

import "testing"

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "https URL", value: "https://hooks.example.com/webhook", valid: true},
		{name: "http URL with query", value: "http://localhost:8080/notify?token=value", valid: true},
		{name: "ipv6 URL", value: "http://[::1]:8080/notify", valid: true},
		{name: "empty", value: "", valid: false},
		{name: "unsupported scheme", value: "redis://localhost:6379", valid: false},
		{name: "missing host", value: "https:///notify", valid: false},
		{name: "relative URL", value: "example.com/notify", valid: false},
		{name: "userinfo", value: "https://user:password@example.com/notify", valid: false},
		{name: "fragment", value: "https://example.com/notify#fragment", valid: false},
		{name: "whitespace", value: "https://example.com/my webhook", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPURL(tt.value)
			if (err == nil) != tt.valid {
				t.Fatalf("ValidateHTTPURL(%q) error = %v, valid = %t", tt.value, err, tt.valid)
			}
		})
	}
}
