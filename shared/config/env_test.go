package config

import (
	"strings"
	"testing"
	"time"
)

func TestStringAndLoaderRequiredString(t *testing.T) {
	t.Setenv("NAVIRE_TEST_STRING", "value")
	loader := NewLoader()
	if got := loader.String("NAVIRE_TEST_STRING", "fallback"); got != "value" {
		t.Fatalf("String() = %q, want value", got)
	}
	t.Setenv("NAVIRE_TEST_EMPTY", "")
	if got := loader.String("NAVIRE_TEST_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("String(empty) = %q, want fallback", got)
	}
	if got := loader.RequiredString("NAVIRE_TEST_MISSING"); got != "" {
		t.Fatalf("RequiredString() = %q, want empty fallback", got)
	}
	if err := loader.Err(); err == nil {
		t.Fatal("Loader() ignored a missing required variable")
	}
}

func TestTypedValuesRejectInvalidInput(t *testing.T) {
	loader := NewLoader()
	t.Setenv("NAVIRE_TEST_INT", "not-an-int")
	loader.Int("NAVIRE_TEST_INT", 1)
	if err := loader.Err(); err == nil || !strings.Contains(err.Error(), "NAVIRE_TEST_INT") {
		t.Fatalf("Int() error = %v, want named error", err)
	}

	loader = NewLoader()
	t.Setenv("NAVIRE_TEST_BOOL", "not-a-bool")
	loader.Bool("NAVIRE_TEST_BOOL", true)
	if err := loader.Err(); err == nil {
		t.Fatal("Bool() accepted invalid value")
	}

	loader = NewLoader()
	t.Setenv("NAVIRE_TEST_DURATION", "3seconds")
	loader.Duration("NAVIRE_TEST_DURATION", time.Second)
	if err := loader.Err(); err == nil {
		t.Fatal("Duration() accepted invalid value")
	}

	loader = NewLoader()
	t.Setenv("NAVIRE_TEST_DURATION", "30d")
	if got := loader.Duration("NAVIRE_TEST_DURATION", time.Second); got != 30*24*time.Hour {
		t.Fatalf("Duration(30d) = %s, want 720h", got)
	}
}

func TestNormalizeListenAddr(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{input: "8090", want: ":8090"},
		{input: ":8090", want: ":8090"},
		{input: "127.0.0.1:8090", want: "127.0.0.1:8090"},
	} {
		if got := NormalizeListenAddr(test.input); got != test.want {
			t.Fatalf("NormalizeListenAddr(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestNormalizeLogLevel(t *testing.T) {
	for _, test := range []struct {
		input, want string
	}{
		{input: "debug", want: "debug"},
		{input: " info ", want: "info"},
		{input: "warning", want: "warn"},
		{input: "error", want: "error"},
	} {
		got, ok := NormalizeLogLevel(test.input)
		if !ok || got != test.want {
			t.Fatalf("NormalizeLogLevel(%q) = %q, %t, want %q, true", test.input, got, ok, test.want)
		}
	}
	if _, ok := NormalizeLogLevel("trace"); ok {
		t.Fatal("NormalizeLogLevel(trace) accepted an unsupported level")
	}
}

func TestValidateLogLevel(t *testing.T) {
	if err := ValidateLogLevel("NAVIRE_LOG_LEVEL", "trace"); err == nil {
		t.Fatal("ValidateLogLevel accepted an unsupported level")
	}
	if err := ValidateLogLevel("NAVIRE_LOG_LEVEL", "warning"); err != nil {
		t.Fatalf("ValidateLogLevel rejected warning alias: %v", err)
	}
}
