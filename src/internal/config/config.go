package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default configuration values.
const (
	DefaultAPIURL             = "https://api.neodigit.net/v1"
	DefaultResolvers          = "8.8.8.8,1.1.1.1"
	DefaultPropagationTimeout = 120 * time.Second
	DefaultPollInterval       = 5 * time.Second
	DefaultLogLevel           = "info"
	DefaultLogFormat          = "text"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	APIToken           string
	APIURL             string
	DNSResolvers       []string
	PropagationTimeout time.Duration
	PollInterval       time.Duration
	LogLevel           slog.Level
	LogFormat          string
}

// Load reads configuration from environment variables and returns a validated Config.
func Load() (*Config, error) {
	token := os.Getenv("TECNOCRATICA_API_TOKEN")
	if token == "" {
		return nil, errors.New("TECNOCRATICA_API_TOKEN is required")
	}

	apiURL := envOrDefault("TECNOCRATICA_API_URL", DefaultAPIURL)

	resolversStr := envOrDefault("TECNOCRATICA_DNS_RESOLVERS", DefaultResolvers)
	resolvers := parseResolvers(resolversStr)
	if len(resolvers) == 0 {
		return nil, errors.New("TECNOCRATICA_DNS_RESOLVERS must contain at least one resolver")
	}

	propagationTimeout, err := parseDurationSeconds("TECNOCRATICA_PROPAGATION_TIMEOUT", DefaultPropagationTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid TECNOCRATICA_PROPAGATION_TIMEOUT: %w", err)
	}

	pollInterval, err := parseDurationSeconds("TECNOCRATICA_POLL_INTERVAL", DefaultPollInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid TECNOCRATICA_POLL_INTERVAL: %w", err)
	}

	logLevel, err := parseLogLevel(envOrDefault("TECNOCRATICA_LOG_LEVEL", DefaultLogLevel))
	if err != nil {
		return nil, fmt.Errorf("invalid TECNOCRATICA_LOG_LEVEL: %w", err)
	}

	logFormat := strings.ToLower(envOrDefault("TECNOCRATICA_LOG_FORMAT", DefaultLogFormat))
	if logFormat != "text" && logFormat != "json" {
		return nil, fmt.Errorf("invalid TECNOCRATICA_LOG_FORMAT: must be 'text' or 'json', got %q", logFormat)
	}

	return &Config{
		APIToken:           token,
		APIURL:             apiURL,
		DNSResolvers:       resolvers,
		PropagationTimeout: propagationTimeout,
		PollInterval:       pollInterval,
		LogLevel:           logLevel,
		LogFormat:          logFormat,
	}, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseResolvers(s string) []string {
	var resolvers []string
	for r := range strings.SplitSeq(s, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			resolvers = append(resolvers, r)
		}
	}
	return resolvers
}

func parseDurationSeconds(envKey string, defaultVal time.Duration) (time.Duration, error) {
	s := os.Getenv(envKey)
	if s == "" {
		return defaultVal, nil
	}
	secs, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("must be an integer (seconds), got %q", s)
	}
	if secs <= 0 {
		return 0, fmt.Errorf("must be a positive integer, got %d", secs)
	}
	return time.Duration(secs) * time.Second, nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("must be debug/info/warn/error, got %q", s)
	}
}
