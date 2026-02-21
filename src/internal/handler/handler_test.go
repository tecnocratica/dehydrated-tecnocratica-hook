package handler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/dns"
)

// mockProvider implements dns.DNSProvider for testing.
type mockProvider struct {
	appendRecords []libdns.Record
	deleteRecords []libdns.Record
	appendErr     error
	deleteErr     error
	appendCalls   int
	deleteCalls   int
}

func (m *mockProvider) AppendRecords(_ context.Context, _ string, recs []libdns.Record) ([]libdns.Record, error) {
	m.appendCalls++
	m.appendRecords = append(m.appendRecords, recs...)
	if m.appendErr != nil {
		return nil, m.appendErr
	}
	return recs, nil
}

func (m *mockProvider) DeleteRecords(_ context.Context, _ string, recs []libdns.Record) ([]libdns.Record, error) {
	m.deleteCalls++
	m.deleteRecords = append(m.deleteRecords, recs...)
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	return recs, nil
}

// mockResolver for propagation checking in handler tests.
type mockHandlerResolver struct {
	responses map[string][]string
}

func (m *mockHandlerResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	if txts, ok := m.responses[name]; ok {
		return txts, nil
	}
	return nil, nil
}

func newTestHandler(provider *mockProvider, resolverResponses map[string][]string) *Handler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	resolver := &mockHandlerResolver{responses: resolverResponses}

	return &Handler{
		Provider: provider,
		Checker: &dns.PropagationChecker{
			Resolvers: []dns.Resolver{resolver},
			Timeout:   5 * time.Second,
			Interval:  50 * time.Millisecond,
			Logger:    logger,
		},
		Logger: logger,
	}
}

func TestHandle_DeployChallenge(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, map[string][]string{
		"_acme-challenge.example.com.": {"test-token"},
	})

	err := h.Handle(context.Background(), "deploy_challenge",
		[]string{"example.com", "token-file", "test-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.appendCalls != 1 {
		t.Errorf("appendCalls = %d, want 1", provider.appendCalls)
	}
	if len(provider.appendRecords) != 1 {
		t.Fatalf("appendRecords len = %d, want 1", len(provider.appendRecords))
	}
	rec := provider.appendRecords[0]
	rr := rec.RR()
	if rr.Type != "TXT" {
		t.Errorf("record type = %q, want TXT", rr.Type)
	}
	if rr.Name != "_acme-challenge" {
		t.Errorf("record name = %q, want _acme-challenge", rr.Name)
	}
	if rr.Data != "test-token" {
		t.Errorf("record data = %q, want test-token", rr.Data)
	}
}

func TestHandle_DeployChallenge_HookChain(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, map[string][]string{
		"_acme-challenge.one.example.com.": {"token-1"},
		"_acme-challenge.two.example.com.": {"token-2"},
	})

	err := h.Handle(context.Background(), "deploy_challenge",
		[]string{
			"one.example.com", "tf1", "token-1",
			"two.example.com", "tf2", "token-2",
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.appendCalls != 2 {
		t.Errorf("appendCalls = %d, want 2", provider.appendCalls)
	}
}

func TestHandle_DeployChallenge_Wildcard(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, map[string][]string{
		"_acme-challenge.example.com.": {"wildcard-token"},
	})

	err := h.Handle(context.Background(), "deploy_challenge",
		[]string{"*.example.com", "token-file", "wildcard-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(provider.appendRecords) != 1 {
		t.Fatalf("appendRecords len = %d, want 1", len(provider.appendRecords))
	}
	if provider.appendRecords[0].RR().Name != "_acme-challenge" {
		t.Errorf("record name = %q, want _acme-challenge", provider.appendRecords[0].RR().Name)
	}
}

func TestHandle_DeployChallenge_ProviderError(t *testing.T) {
	provider := &mockProvider{appendErr: errors.New("API error")}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "deploy_challenge",
		[]string{"example.com", "tf", "tv"})
	if err == nil {
		t.Fatal("expected error from provider")
	}
}

func TestHandle_DeployChallenge_InvalidArgs(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "deploy_challenge", []string{"only-one-arg"})
	if err == nil {
		t.Fatal("expected error for invalid args")
	}
}

func TestHandle_CleanChallenge(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "clean_challenge",
		[]string{"example.com", "token-file", "test-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", provider.deleteCalls)
	}
}

func TestHandle_CleanChallenge_ProviderError(t *testing.T) {
	provider := &mockProvider{deleteErr: errors.New("API error")}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "clean_challenge",
		[]string{"example.com", "tf", "tv"})
	if err == nil {
		t.Fatal("expected error from provider")
	}
}

func TestHandle_CleanChallenge_InvalidArgs(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "clean_challenge", []string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestHandle_StartupHook(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "startup_hook", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandle_ExitHook(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "exit_hook", []string{"success"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandle_ExitHookNoArgs(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "exit_hook", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandle_NoOpHooks(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	noOpHooks := []string{
		"deploy_cert", "deploy_ocsp", "unchanged_cert",
		"invalid_challenge", "request_failure", "sync_cert", "generate_csr",
	}

	for _, hook := range noOpHooks {
		t.Run(hook, func(t *testing.T) {
			err := h.Handle(context.Background(), hook, nil)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", hook, err)
			}
		})
	}
}

func TestHandle_UnknownHook(t *testing.T) {
	provider := &mockProvider{}
	h := newTestHandler(provider, nil)

	err := h.Handle(context.Background(), "totally_unknown_hook", nil)
	if err != nil {
		t.Fatalf("unknown hooks should not return error, got: %v", err)
	}
}
