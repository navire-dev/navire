package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/navire-dev/navire/shared/providers"
)

func TestLoadTargetsFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetsPath := filepath.Join(tmpDir, "gotify.yml")

	content := `provider: gotify
enabled: true
endpoints:
  - key: prod
    enabled: true
    url: https://gotify.example.com/message
    auth:
      type: query
      param: token
      value: test_token
`

	if err := os.WriteFile(targetsPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tf, err := LoadTargetsFile(targetsPath)
	if err != nil {
		t.Fatalf("LoadTargetsFile failed: %v", err)
	}

	if tf.Provider != providers.Gotify {
		t.Errorf("expected provider 'gotify', got %q", tf.Provider)
	}

	if len(tf.Endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(tf.Endpoints))
	}
	if tf.Endpoints[0].Key != "prod" {
		t.Errorf("expected endpoint key 'prod', got %q", tf.Endpoints[0].Key)
	}
}

func TestLoadAllTargets(t *testing.T) {
	tmpDir := t.TempDir()

	gotifyContent := `provider: gotify
enabled: true
endpoints:
  - key: prod
    enabled: true
    url: https://gotify.example.com/message
    auth:
      type: query
      param: token
      value: token
`
	if err := os.WriteFile(filepath.Join(tmpDir, "gotify.yml"), []byte(gotifyContent), 0o644); err != nil {
		t.Fatal(err)
	}

	endpoints, err := LoadAllTargets(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllTargets failed: %v", err)
	}

	if len(endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(endpoints))
	}

	gotifyProd, exists := endpoints["gotify_prod"]
	if !exists {
		t.Fatal("expected endpoint 'gotify_prod' to exist")
	}

	if gotifyProd.Provider != providers.Gotify {
		t.Errorf("expected provider 'gotify', got %q", gotifyProd.Provider)
	}

	if gotifyProd.FullName != "gotify_prod" {
		t.Errorf("expected full_name 'gotify_prod', got %q", gotifyProd.FullName)
	}
}

func TestLoadAllTargetsRejectsWholeConfigurationWhenOneFileIsInvalid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	valid := `provider: gotify
enabled: true
endpoints:
  - key: prod
    enabled: true
    url: https://gotify.example.com/message
    auth:
      type: none
`
	invalid := `provider: unknown
enabled: true
endpoints: []
`

	if err := os.WriteFile(filepath.Join(tmpDir, "gotify.yml"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}

	endpoints, err := LoadAllTargets(tmpDir)
	if err == nil {
		t.Fatal("expected the complete configuration to be rejected")
	}
	if endpoints != nil {
		t.Fatalf("expected no partial endpoints, got %d", len(endpoints))
	}
}

func TestLoadTargetsFileRejectsInvalidEndpointKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "gotify.yml")
	content := `provider: gotify
enabled: true
endpoints:
  - key: Invalid Endpoint
    enabled: true
    url: https://gotify.example.com/message
    auth:
      type: none
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadTargetsFile(path); err == nil {
		t.Fatal("LoadTargetsFile() accepted an invalid endpoint key")
	}
}

func TestLoadAllTargetsLoadsActiveProviders(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gotify := `provider: gotify
enabled: true
endpoints:
  - key: prod
    enabled: true
    url: https://gotify.example.com/message
    auth:
      type: none
`
	slack := `provider: slack
enabled: true
endpoints:
  - key: cicd
    enabled: true
    url: https://hooks.slack.com/services/test
    auth:
      type: none
`
	if err := os.WriteFile(filepath.Join(tmpDir, "gotify.yml"), []byte(gotify), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "slack.yml"), []byte(slack), 0o644); err != nil {
		t.Fatal(err)
	}

	endpoints, err := LoadAllTargets(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllTargets() error = %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("LoadAllTargets() returned %d endpoints, want 2", len(endpoints))
	}
	if _, ok := endpoints["gotify_prod"]; !ok {
		t.Fatal("LoadAllTargets() did not ingest the active Gotify provider")
	}
	if _, ok := endpoints["slack_cicd"]; !ok {
		t.Fatal("LoadAllTargets() did not ingest the active Slack provider")
	}
}

func TestEmptyConfigurationIsValid(t *testing.T) {
	t.Parallel()

	endpoints, err := loadAllTargets(t.TempDir(), false)
	if err != nil {
		t.Fatalf("expected empty runtime configuration to be valid: %v", err)
	}
	if len(endpoints) != 0 {
		t.Fatalf("expected no endpoints, got %d", len(endpoints))
	}
}
