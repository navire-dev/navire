package validation

import "testing"

func TestIsKey(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "letters and digits", value: "fake123", valid: true},
		{name: "hyphen and underscore", value: "fake_programmatic-v2", valid: true},
		{name: "empty", value: "", valid: false},
		{name: "uppercase", value: "Fake", valid: false},
		{name: "space", value: "fake programmatic", valid: false},
		{name: "dot", value: "fake.programmatic", valid: false},
		{name: "leading hyphen", value: "-fake", valid: false},
		{name: "maximum length", value: "12345678901234567890123456789012345678901234567890", valid: true},
		{name: "too long", value: "123456789012345678901234567890123456789012345678901", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKey(tt.value); got != tt.valid {
				t.Fatalf("IsKey(%q) = %t, want %t", tt.value, got, tt.valid)
			}
		})
	}
}
