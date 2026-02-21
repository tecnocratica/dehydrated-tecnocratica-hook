package dns

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	tecnocratica "github.com/libdns/tecnocratica"
)

// Compile-time interface guard: tecnocratica.Provider must satisfy DNSProvider.
var _ DNSProvider = (*tecnocratica.Provider)(nil)

// mockResolver is a test double for Resolver.
type mockResolver struct {
	responses map[string][]string
	err       error
	callCount int
}

func (m *mockResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	if txts, ok := m.responses[name]; ok {
		return txts, nil
	}
	return nil, nil
}

// delayedMockResolver returns empty results until a call threshold, then returns data.
type delayedMockResolver struct {
	responses map[string][]string
	threshold int
	callCount int
}

func (m *delayedMockResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	m.callCount++
	if m.callCount >= m.threshold {
		if txts, ok := m.responses[name]; ok {
			return txts, nil
		}
	}
	return nil, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestWaitForPropagation_ImmediateSuccess(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"test-token-123"},
		},
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   10 * time.Second,
		Interval:  100 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "test-token-123"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForPropagation_MultipleResolvers(t *testing.T) {
	r1 := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"test-token"},
		},
	}
	r2 := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"test-token"},
		},
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{r1, r2},
		Timeout:   10 * time.Second,
		Interval:  100 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "test-token"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForPropagation_DelayedSuccess(t *testing.T) {
	resolver := &delayedMockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"delayed-token"},
		},
		threshold: 3,
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   10 * time.Second,
		Interval:  50 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "delayed-token"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolver.callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", resolver.callCount)
	}
}

func TestWaitForPropagation_Timeout(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{}, // never returns the token
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   300 * time.Millisecond,
		Interval:  100 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "never-found"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, nil) && err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestWaitForPropagation_ContextCancelled(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{}, // never returns the token
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   30 * time.Second,
		Interval:  50 * time.Millisecond,
		Logger:    testLogger(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "token"},
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(120 * time.Millisecond)
		cancel()
	}()

	err := checker.WaitForPropagation(ctx, records)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestWaitForPropagation_ResolverError(t *testing.T) {
	resolver := &mockResolver{
		err: errors.New("network error"),
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   300 * time.Millisecond,
		Interval:  100 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.example.com.", Token: "token"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err == nil {
		t.Fatal("expected error due to resolver failures")
	}
}

func TestWaitForPropagation_MultipleRecords(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.one.example.com.": {"token-1"},
			"_acme-challenge.two.example.com.": {"token-2"},
		},
	}

	checker := &PropagationChecker{
		Resolvers: []Resolver{resolver},
		Timeout:   10 * time.Second,
		Interval:  100 * time.Millisecond,
		Logger:    testLogger(),
	}

	records := []ChallengeRecord{
		{FQDN: "_acme-challenge.one.example.com.", Token: "token-1"},
		{FQDN: "_acme-challenge.two.example.com.", Token: "token-2"},
	}

	err := checker.WaitForPropagation(context.Background(), records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTXT_Found(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"other-value", "my-token"},
		},
	}

	found, err := checkTXT(context.Background(), resolver, "_acme-challenge.example.com.", "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected token to be found")
	}
}

func TestCheckTXT_NotFound(t *testing.T) {
	resolver := &mockResolver{
		responses: map[string][]string{
			"_acme-challenge.example.com.": {"other-value"},
		},
	}

	found, err := checkTXT(context.Background(), resolver, "_acme-challenge.example.com.", "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected token to not be found")
	}
}

func TestCheckTXT_Error(t *testing.T) {
	resolver := &mockResolver{
		err: errors.New("lookup failed"),
	}

	_, err := checkTXT(context.Background(), resolver, "_acme-challenge.example.com.", "token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewNetResolver(t *testing.T) {
	// Verify constructor doesn't panic
	r := NewNetResolver("8.8.8.8")
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}

	r2 := NewNetResolver("8.8.8.8:53")
	if r2 == nil {
		t.Fatal("expected non-nil resolver")
	}
}
