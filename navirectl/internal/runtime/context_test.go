package runtime

import "testing"

func TestNewCopiesArguments(t *testing.T) {
	args := []string{"validate", "--strict"}
	ctx := New(args, nil, nil)
	args[0] = "version"

	if ctx.Args[0] != "validate" {
		t.Fatalf("runtime context retained mutable argument slice: %q", ctx.Args)
	}
}

func TestNewPreservesWriters(t *testing.T) {
	writer := testWriter{}
	errWriter := testWriter{}
	ctx := New(nil, writer, errWriter)

	if ctx.Writer != writer {
		t.Fatal("runtime context did not preserve output writer")
	}
	if ctx.ErrWriter != errWriter {
		t.Fatal("runtime context did not preserve error writer")
	}
}

type testWriter struct{}

func (testWriter) Write([]byte) (int, error) { return 0, nil }
