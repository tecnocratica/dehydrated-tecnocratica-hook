package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/config"
)

func TestRun_NoArgs(t *testing.T) {
	code := run([]string{"hook"})
	if code != 1 {
		t.Errorf("run() = %d, want 1 for missing hook argument", code)
	}
}

func TestRun_MissingToken(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "")
	code := run([]string{"hook", "startup_hook"})
	if code != 1 {
		t.Errorf("run() = %d, want 1 for missing API token", code)
	}
}

func TestRun_StartupHook(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	code := run([]string{"hook", "startup_hook"})
	if code != 0 {
		t.Errorf("run() = %d, want 0 for startup_hook", code)
	}
}

func TestRun_ExitHook(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	code := run([]string{"hook", "exit_hook", "done"})
	if code != 0 {
		t.Errorf("run() = %d, want 0 for exit_hook", code)
	}
}

func TestRun_UnknownHook(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	code := run([]string{"hook", "some_unknown_hook"})
	if code != 0 {
		t.Errorf("run() = %d, want 0 for unknown hook", code)
	}
}

func TestRun_NoOpHook(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	code := run([]string{"hook", "deploy_cert"})
	if code != 0 {
		t.Errorf("run() = %d, want 0 for no-op hook", code)
	}
}

func TestRun_DeployChallengeInvalidArgs(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	// deploy_challenge with insufficient args should fail
	code := run([]string{"hook", "deploy_challenge", "only-one"})
	if code != 1 {
		t.Errorf("run() = %d, want 1 for invalid deploy_challenge args", code)
	}
}

func TestSetupLogger_TextFormat(t *testing.T) {
	cfg := &config.Config{
		LogLevel:  slog.LevelInfo,
		LogFormat: "text",
	}
	logger := setupLogger(cfg)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestSetupLogger_JSONFormat(t *testing.T) {
	cfg := &config.Config{
		LogLevel:  slog.LevelDebug,
		LogFormat: "json",
	}
	logger := setupLogger(cfg)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestRun_CustomConfig(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_API_URL", "https://api.example.com/v1")
	t.Setenv("TECNOCRATICA_DNS_RESOLVERS", "1.1.1.1")
	t.Setenv("TECNOCRATICA_LOG_LEVEL", "debug")
	t.Setenv("TECNOCRATICA_LOG_FORMAT", "json")

	code := run([]string{"hook", "startup_hook"})
	if code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}
}

func TestMain(m *testing.M) {
	// Ensure we don't accidentally use real credentials
	_ = os.Unsetenv("TECNOCRATICA_API_TOKEN")
	os.Exit(m.Run())
}
