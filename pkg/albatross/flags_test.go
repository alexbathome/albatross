package albatross

import (
	"strings"
	"testing"
	"time"
)

func TestFlagFromArgs(t *testing.T) {
	fs := NewFlagSet("test")
	foo := Flag(fs, fs.String, "foo", "", "the foo flag", "", false)

	if err := fs.Parse([]string{"--foo", "ok"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := foo(); got != "ok" {
		t.Fatalf("foo() = %q, want %q", got, "ok")
	}
}

func TestFlagDefault(t *testing.T) {
	fs := NewFlagSet("test")
	foo := Flag(fs, fs.String, "foo", "fallback", "the foo flag", "", false)

	if err := fs.Parse(nil); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := foo(); got != "fallback" {
		t.Fatalf("foo() = %q, want %q", got, "fallback")
	}
}

func TestFlagEnvOverridesArgs(t *testing.T) {
	t.Setenv("FOO_ENV", "from-env")

	fs := NewFlagSet("test")
	foo := Flag(fs, fs.String, "foo", "", "the foo flag", "FOO_ENV", false)

	if err := fs.Parse([]string{"--foo", "from-args"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := foo(); got != "from-env" {
		t.Fatalf("foo() = %q, want %q", got, "from-env")
	}
}

func TestFlagEnvUnsetKeepsArgs(t *testing.T) {
	fs := NewFlagSet("test")
	foo := Flag(fs, fs.String, "foo", "", "the foo flag", "FOO_ENV_UNSET", false)

	if err := fs.Parse([]string{"--foo", "from-args"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := foo(); got != "from-args" {
		t.Fatalf("foo() = %q, want %q", got, "from-args")
	}
}

func TestFlagTypedEnvOverrides(t *testing.T) {
	t.Setenv("PORT_ENV", "9090")
	t.Setenv("DEBUG_ENV", "true")
	t.Setenv("TIMEOUT_ENV", "1m30s")

	fs := NewFlagSet("test")
	port := Flag(fs, fs.Int, "port", 8080, "listen port", "PORT_ENV", false)
	debug := Flag(fs, fs.Bool, "debug", false, "debug mode", "DEBUG_ENV", false)
	timeout := Flag(fs, fs.Duration, "timeout", time.Second, "timeout", "TIMEOUT_ENV", false)

	if err := fs.Parse(nil); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := port(); got != 9090 {
		t.Fatalf("port() = %d, want 9090", got)
	}
	if got := debug(); !got {
		t.Fatal("debug() = false, want true")
	}
	if got := timeout(); got != 90*time.Second {
		t.Fatalf("timeout() = %v, want 1m30s", got)
	}
}

func TestFlagInvalidEnvValue(t *testing.T) {
	t.Setenv("PORT_ENV", "not-a-number")

	fs := NewFlagSet("test")
	_ = Flag(fs, fs.Int, "port", 8080, "listen port", "PORT_ENV", false)

	err := fs.Parse(nil)
	if err == nil {
		t.Fatal("expected an error for a non-numeric env value")
	}
	if !strings.Contains(err.Error(), "PORT_ENV") {
		t.Fatalf("error should name the env var, got: %v", err)
	}
}

func TestRequiredFlagMissing(t *testing.T) {
	fs := NewFlagSet("test")
	_ = Flag(fs, fs.String, "foo", "", "the foo flag", "FOO_ENV_UNSET", true)

	err := fs.Parse(nil)
	if err == nil {
		t.Fatal("expected an error for a missing required flag")
	}
	if !strings.Contains(err.Error(), "-foo") || !strings.Contains(err.Error(), "FOO_ENV_UNSET") {
		t.Fatalf("error should name the flag and env var, got: %v", err)
	}
}

func TestRequiredFlagFromEnvOnly(t *testing.T) {
	t.Setenv("FOO_ENV", "from-env")

	fs := NewFlagSet("test")
	foo := Flag(fs, fs.String, "foo", "", "the foo flag", "FOO_ENV", true)

	if err := fs.Parse(nil); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if got := foo(); got != "from-env" {
		t.Fatalf("foo() = %q, want %q", got, "from-env")
	}
}

func TestRequiredFlagExplicitZeroValue(t *testing.T) {
	fs := NewFlagSet("test")
	port := Flag(fs, fs.Int, "port", 8080, "listen port", "", true)

	if err := fs.Parse([]string{"--port", "0"}); err != nil {
		t.Fatalf("an explicitly provided zero value should satisfy required: %v", err)
	}
	if got := port(); got != 0 {
		t.Fatalf("port() = %d, want 0", got)
	}
}

func TestRequiredDefaultDoesNotSatisfy(t *testing.T) {
	fs := NewFlagSet("test")
	_ = Flag(fs, fs.String, "foo", "has-default", "the foo flag", "", true)

	if err := fs.Parse(nil); err == nil {
		t.Fatal("a default value should not satisfy a required flag")
	}
}
