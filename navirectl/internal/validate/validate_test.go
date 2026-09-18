package validate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/navire-dev/navire/navirectl/internal/config"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
	"gopkg.in/yaml.v3"
)

type testLogger struct {
	errors    []string
	successes []string
}

func (l *testLogger) Info(string, ...any) {}
func (l *testLogger) Success(message string, args ...any) {
	l.successes = append(l.successes, fmt.Sprintf(message, args...))
}
func (l *testLogger) Warn(string, ...any) {}
func (l *testLogger) Error(message string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(message, args...))
}
func (l *testLogger) Debug(string, ...any) {}

func TestTemplatesValidatesConfiguredDirectory(t *testing.T) {
	root := t.TempDir()
	templates := filepath.Join(root, "templates")
	if err := os.Mkdir(templates, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(templates, "nested", "team")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("key: example\nvariants:\n  ok:\n    title: OK\n    body: Done\n    state: success\n    priority: normal\nproviders:\n  gotify:\n    endpoints:\n      prod: {}\n")
	if err := os.WriteFile(filepath.Join(templates, "example.yml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	nestedContent := []byte("key: nested\nvariants:\n  ok:\n    title: OK\n    body: Done\n    state: success\n    priority: normal\nproviders:\n  gotify:\n    endpoints:\n      prod: {}\n")
	if err := os.WriteFile(filepath.Join(nested, "nested.yml"), nestedContent, 0o644); err != nil {
		t.Fatal(err)
	}

	logger := &testLogger{}
	rt := runtime.New(nil, io.Discard, io.Discard)
	rt.Logger = logger
	rt.Values[config.RuntimeConfigKey] = config.Config{SchemaVersion: 1, TemplatesPath: "templates", ProvidersPath: "providers"}
	rt.Values[config.RuntimeRootKey] = root

	if err := Templates(context.Background(), rt); err != nil {
		t.Fatalf("Templates() error = %v; logged errors = %v", err, logger.errors)
	}
	if len(logger.errors) != 0 {
		t.Fatalf("logger errors = %v, want none", logger.errors)
	}
	wantSuccesses := []string{
		"✓ template example is valid",
		"✓ template nested is valid",
		"validation complete: 2 template(s) validated",
	}
	if len(logger.successes) != len(wantSuccesses) ||
		logger.successes[0] != wantSuccesses[0] ||
		logger.successes[1] != wantSuccesses[1] ||
		logger.successes[2] != wantSuccesses[2] {
		t.Fatalf("logger successes = %v, want validation of two templates", logger.successes)
	}
}

func TestHumanizeErrorUnknownYAMLField(t *testing.T) {
	err := &yaml.TypeError{Errors: []string{
		"line 2: field tttt not found in type template.Definition",
	}}

	if got, want := humanizeError(err), `unknown field "tttt" (line 2)`; got != want {
		t.Fatalf("humanizeError() = %q, want %q", got, want)
	}
}
