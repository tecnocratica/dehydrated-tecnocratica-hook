package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tecnocratica "github.com/libdns/tecnocratica"
	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/config"
	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/dns"
	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/handler"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "dehydrated-tecnocratica-hook %s (%s, %s)\n", version, commit, date)
		fmt.Fprintf(os.Stderr, "Usage: %s <hook> [args...]\n", args[0])
		return 1
	}

	hook := args[1]
	hookArgs := args[2:]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		return 1
	}

	logger := setupLogger(cfg)

	logger.Debug("hook invoked",
		"hook", hook,
		"args", hookArgs,
		"version", version,
	)

	provider := &tecnocratica.Provider{
		APIToken: cfg.APIToken,
		APIURL:   cfg.APIURL,
	}

	var resolvers []dns.Resolver
	for _, addr := range cfg.DNSResolvers {
		resolvers = append(resolvers, dns.NewNetResolver(addr))
	}

	checker := &dns.PropagationChecker{
		Resolvers: resolvers,
		Timeout:   cfg.PropagationTimeout,
		Interval:  cfg.PollInterval,
		Logger:    logger,
	}

	h := &handler.Handler{
		Provider: provider,
		Checker:  checker,
		Logger:   logger,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := h.Handle(ctx, hook, hookArgs); err != nil {
		logger.Error("hook failed", "hook", hook, "error", err)
		return 1
	}

	return 0
}

func setupLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}

	var h slog.Handler
	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(h)
}
