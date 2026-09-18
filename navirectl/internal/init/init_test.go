package init

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/navire-dev/navire/navirectl/internal/runtime"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)    {}
func (testLogger) Success(string, ...any) {}
func (testLogger) Warn(string, ...any)    {}
func (testLogger) Error(string, ...any)   {}
func (testLogger) Debug(string, ...any)   {}

func TestNewCreatesAndValidatesRepositoryConfiguration(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	var output bytes.Buffer
	rt := runtime.New(nil, &output, &output)
	rt.Reader = strings.NewReader("")
	rt.Logger = testLogger{}

	if err := New(context.Background(), rt); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := os.Stat(".config.json"); err != nil {
		t.Fatalf(".config.json was not created: %v", err)
	}
	for _, path := range []string{"templates", "providers"} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s directory was not created: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", path)
		}
	}
	if !strings.Contains(output.String(), "Repository configuration:") {
		t.Fatalf("output = %q, want configuration summary", output.String())
	}
}
