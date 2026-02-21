package dns

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// DNSProvider abstracts DNS record operations. The tecnocratica.Provider
// satisfies this interface directly via its libdns implementation.
type DNSProvider interface {
	AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error)
	DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error)
}

// ChallengeRecord represents a DNS TXT record that must be verified.
type ChallengeRecord struct {
	FQDN  string // e.g. "_acme-challenge.example.com."
	Token string // expected TXT value
}

// Resolver abstracts DNS TXT lookups for testing.
type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// NetResolver wraps net.Resolver to query a specific DNS server.
type NetResolver struct {
	resolver *net.Resolver
}

// NewNetResolver creates a Resolver that queries the given DNS server address (e.g. "8.8.8.8").
func NewNetResolver(server string) *NetResolver {
	addr := server
	if !strings.Contains(addr, ":") {
		addr += ":53"
	}
	return &NetResolver{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", addr)
			},
		},
	}
}

// LookupTXT performs a TXT record lookup using the configured DNS server.
func (r *NetResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.resolver.LookupTXT(ctx, name)
}

// PropagationChecker verifies that DNS TXT records have propagated across resolvers.
type PropagationChecker struct {
	Resolvers []Resolver
	Timeout   time.Duration
	Interval  time.Duration
	Logger    *slog.Logger
}

// WaitForPropagation polls all resolvers until every challenge record is found
// or the timeout is exceeded. The polling interval grows by 1.5x each iteration.
func (pc *PropagationChecker) WaitForPropagation(ctx context.Context, records []ChallengeRecord) error {
	deadline := time.Now().Add(pc.Timeout)
	interval := pc.Interval

	for {
		allFound := true
		for _, rec := range records {
			for i, resolver := range pc.Resolvers {
				found, err := checkTXT(ctx, resolver, rec.FQDN, rec.Token)
				if err != nil {
					pc.Logger.Debug("DNS lookup error",
						"fqdn", rec.FQDN,
						"resolver_index", i,
						"error", err,
					)
					allFound = false
					continue
				}
				if !found {
					pc.Logger.Debug("TXT record not yet propagated",
						"fqdn", rec.FQDN,
						"resolver_index", i,
					)
					allFound = false
				}
			}
		}

		if allFound {
			pc.Logger.Info("all DNS records propagated successfully")
			return nil
		}

		if time.Now().Add(interval).After(deadline) {
			return fmt.Errorf("propagation timeout after %s: not all records found", pc.Timeout)
		}

		pc.Logger.Debug("waiting for propagation", "interval", interval)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(interval):
		}

		// Exponential backoff: interval *= 1.5
		interval = time.Duration(float64(interval) * 1.5)
	}
}

// checkTXT looks up TXT records for fqdn and returns true if token is found.
func checkTXT(ctx context.Context, resolver Resolver, fqdn, token string) (bool, error) {
	txts, err := resolver.LookupTXT(ctx, fqdn)
	if err != nil {
		// DNS NXDOMAIN / no-answer is expected during propagation
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return false, nil
		}
		return false, err
	}
	if slices.Contains(txts, token) {
		return true, nil
	}
	return false, nil
}
