package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogReloadDiscardsInvalidTemplates(t *testing.T) {
	dir := t.TempDir()
	valid := `key: valid
variants:
  ok:
    state: success
    title: ok
    body: ok
providers:
  gotify:
    endpoints:
      prod: {}
`
	invalid := valid + "unexpected: true\n"
	if err := os.WriteFile(filepath.Join(dir, "valid.yml"), []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid.yml"), []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog := NewCatalog(dir)
	issues, err := catalog.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Err.Error(), "unexpected") {
		for _, issue := range issues {
			t.Logf("issue %s: %v", issue.File, issue.Err)
		}
		t.Fatalf("issues = %#v, want one unknown-field issue", issues)
	}
	if _, err := catalog.LoadByKey("valid"); err != nil {
		t.Fatalf("valid template was discarded: %v", err)
	}
	if _, err := catalog.LoadByKey("invalid"); err == nil {
		t.Fatal("invalid template was loaded")
	}
}

func TestCatalogReloadDiscardsTemplateWithInvalidKey(t *testing.T) {
	dir := t.TempDir()
	invalid := `key: Invalid Template Key
variants:
  ok:
    state: success
    title: ok
    body: ok
providers:
  gotify:
    endpoints:
      prod: {}
`
	if err := os.WriteFile(filepath.Join(dir, "invalid-key.yml"), []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog := NewCatalog(dir)
	issues, err := catalog.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Err.Error(), "template key") {
		t.Fatalf("issues = %#v, want invalid template key issue", issues)
	}
	if catalog.Count() != 0 {
		t.Fatalf("Count() = %d, want invalid template to be discarded", catalog.Count())
	}
}
