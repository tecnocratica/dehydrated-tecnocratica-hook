package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoad_RequiresToken(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when TECNOCRATICA_API_TOKEN is empty")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	// Clear any overrides
	t.Setenv("TECNOCRATICA_API_URL", "")
	t.Setenv("TECNOCRATICA_DNS_RESOLVERS", "")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "")
	t.Setenv("TECNOCRATICA_POLL_INTERVAL", "")
	t.Setenv("TECNOCRATICA_LOG_LEVEL", "")
	t.Setenv("TECNOCRATICA_LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIToken != "test-token" {
		t.Errorf("APIToken = %q, want %q", cfg.APIToken, "test-token")
	}
	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, DefaultAPIURL)
	}
	if len(cfg.DNSResolvers) != 2 || cfg.DNSResolvers[0] != "8.8.8.8" || cfg.DNSResolvers[1] != "1.1.1.1" {
		t.Errorf("DNSResolvers = %v, want [8.8.8.8, 1.1.1.1]", cfg.DNSResolvers)
	}
	if cfg.PropagationTimeout != DefaultPropagationTimeout {
		t.Errorf("PropagationTimeout = %v, want %v", cfg.PropagationTimeout, DefaultPropagationTimeout)
	}
	if cfg.PollInterval != DefaultPollInterval {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, DefaultPollInterval)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "text")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "my-token")
	t.Setenv("TECNOCRATICA_API_URL", "https://api.virtualname.net/v1")
	t.Setenv("TECNOCRATICA_DNS_RESOLVERS", "9.9.9.9, 208.67.222.222")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "60")
	t.Setenv("TECNOCRATICA_POLL_INTERVAL", "10")
	t.Setenv("TECNOCRATICA_LOG_LEVEL", "debug")
	t.Setenv("TECNOCRATICA_LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIURL != "https://api.virtualname.net/v1" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "https://api.virtualname.net/v1")
	}
	if len(cfg.DNSResolvers) != 2 || cfg.DNSResolvers[0] != "9.9.9.9" || cfg.DNSResolvers[1] != "208.67.222.222" {
		t.Errorf("DNSResolvers = %v, want [9.9.9.9, 208.67.222.222]", cfg.DNSResolvers)
	}
	if cfg.PropagationTimeout != 60*time.Second {
		t.Errorf("PropagationTimeout = %v, want %v", cfg.PropagationTimeout, 60*time.Second)
	}
	if cfg.PollInterval != 10*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 10*time.Second)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
}

func TestLoad_InvalidPropagationTimeout(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid propagation timeout")
	}
}

func TestLoad_NegativePropagationTimeout(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "-5")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative propagation timeout")
	}
}

func TestLoad_InvalidPollInterval(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "")
	t.Setenv("TECNOCRATICA_POLL_INTERVAL", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid poll interval")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "")
	t.Setenv("TECNOCRATICA_POLL_INTERVAL", "")
	t.Setenv("TECNOCRATICA_LOG_LEVEL", "verbose")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestLoad_InvalidLogFormat(t *testing.T) {
	t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
	t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "")
	t.Setenv("TECNOCRATICA_POLL_INTERVAL", "")
	t.Setenv("TECNOCRATICA_LOG_LEVEL", "")
	t.Setenv("TECNOCRATICA_LOG_FORMAT", "xml")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log format")
	}
}

func TestLoad_AllLogLevels(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Setenv("TECNOCRATICA_API_TOKEN", "test-token")
			t.Setenv("TECNOCRATICA_PROPAGATION_TIMEOUT", "")
			t.Setenv("TECNOCRATICA_POLL_INTERVAL", "")
			t.Setenv("TECNOCRATICA_LOG_LEVEL", tt.input)
			t.Setenv("TECNOCRATICA_LOG_FORMAT", "")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.LogLevel != tt.want {
				t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, tt.want)
			}
		})
	}
}

func TestParseResolvers_EmptyString(t *testing.T) {
	result := parseResolvers("")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty string, got %v", result)
	}
}

func TestParseResolvers_WhitespaceHandling(t *testing.T) {
	result := parseResolvers(" 8.8.8.8 , , 1.1.1.1 ")
	if len(result) != 2 {
		t.Fatalf("expected 2 resolvers, got %d: %v", len(result), result)
	}
	if result[0] != "8.8.8.8" || result[1] != "1.1.1.1" {
		t.Errorf("resolvers = %v, want [8.8.8.8, 1.1.1.1]", result)
	}
}
