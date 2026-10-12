package render

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	got, err := Render("Payment failed", "Reference: {{ .reference }}", map[string]any{"reference": "pay_8f31c2"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Title != "Payment failed" || got.Body != "Reference: pay_8f31c2" {
		t.Fatalf("Render() = %#v", got)
	}
}

func TestRenderRejectsMissingData(t *testing.T) {
	_, err := Render("Title", "Reference: {{ .reference }}", nil)
	if err == nil || !strings.Contains(err.Error(), "map has no entry for key") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}
