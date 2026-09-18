package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"schema_version":1,"templates_path":"config/templates","providers_path":"config/providers"}`)
	if err := os.WriteFile(filepath.Join(root, Filename), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.TemplatesDir(root), filepath.Join(root, "config", "templates"); got != want {
		t.Fatalf("TemplatesDir() = %q, want %q", got, want)
	}
}

func TestLoadRejectsUnsafePath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"schema_version":1,"templates_path":"../templates","providers_path":"providers"}`)
	if err := os.WriteFile(filepath.Join(root, Filename), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(root); err == nil {
		t.Fatal("Load() succeeded for an unsafe path")
	}
}
