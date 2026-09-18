package validation

import "testing"

type testRequest struct {
	Key      string `json:"key" validate:"required,max=50,navirekey"`
	Priority string `json:"priority" validate:"omitempty,oneof=low normal high critical"`
}

func TestStructRejectsPathSeparatorsWithoutChangingInput(t *testing.T) {
	req := testRequest{Key: "../secrets", Priority: "normal"}
	issues := Struct(req)

	if len(issues) != 1 || issues[0].Field != "key" {
		t.Fatalf("issues = %#v, want key issue", issues)
	}
	if req.Key != "../secrets" {
		t.Fatalf("validation changed input to %q", req.Key)
	}
}

func TestDataPreservesTextAndRejectsInvalidKeys(t *testing.T) {
	text := "line one\nline two\n\nkept"
	data := map[string]any{
		"message":    text,
		"bad key":    "must be rejected, not rewritten",
		"bad\nkey":   "must be safely represented",
		"nested":     map[string]any{"value": true},
		"null":       nil,
		"successful": true,
	}

	issues := Data(data, DataLimits{MaxKeys: 10, MaxKeyLen: 64, MaxValueLen: 100})
	if data["message"] != text {
		t.Fatalf("data text was changed: %q", data["message"])
	}

	if !hasIssue(issues, `data["bad key"]`, "invalid_key") {
		t.Fatalf("issues = %#v, want invalid_key", issues)
	}
	if !hasIssue(issues, `data["bad\nkey"]`, "invalid_key") {
		t.Fatalf("issues = %#v, want safely quoted invalid_key", issues)
	}
	if !hasIssue(issues, "data.nested", "unsupported_type") {
		t.Fatalf("issues = %#v, want unsupported_type", issues)
	}
}

func TestDataRejectsTooManyKeys(t *testing.T) {
	data := map[string]any{"one": "1", "two": "2"}
	issues := Data(data, DataLimits{MaxKeys: 1})

	if !hasIssue(issues, "data", "too_many_keys") {
		t.Fatalf("issues = %#v, want too_many_keys", issues)
	}
}

func hasIssue(issues []Issue, field, code string) bool {
	for _, issue := range issues {
		if issue.Field == field && issue.Code == code {
			return true
		}
	}
	return false
}
