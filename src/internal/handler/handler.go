package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/libdns/libdns"
	"github.com/tecnocratica/dehydrated-tecnocratica-hook/src/internal/dns"
)

// Handler dispatches dehydrated hook calls.
type Handler struct {
	Provider dns.DNSProvider
	Checker  *dns.PropagationChecker
	Logger   *slog.Logger
}

// Handle dispatches the hook to the appropriate implementation.
func (h *Handler) Handle(ctx context.Context, hook string, args []string) error {
	switch hook {
	case "deploy_challenge":
		return h.deployChallenge(ctx, args)
	case "clean_challenge":
		return h.cleanChallenge(ctx, args)
	case "startup_hook":
		h.Logger.Info("dehydrated-tecnocratica-hook started")
		return nil
	case "exit_hook":
		if len(args) > 0 && args[0] != "" {
			h.Logger.Info("dehydrated exiting", "reason", args[0])
		} else {
			h.Logger.Info("dehydrated exiting")
		}
		return nil
	case "deploy_cert", "deploy_ocsp", "unchanged_cert",
		"invalid_challenge", "request_failure", "sync_cert", "generate_csr":
		h.Logger.Debug("hook called (no-op)", "hook", hook)
		return nil
	default:
		// Unknown hooks are silently ignored per dehydrated spec
		h.Logger.Debug("unknown hook ignored", "hook", hook)
		return nil
	}
}

func (h *Handler) deployChallenge(ctx context.Context, args []string) error {
	triplets, err := parseTriplets(args)
	if err != nil {
		return fmt.Errorf("deploy_challenge: %w", err)
	}

	var challengeRecords []dns.ChallengeRecord

	for _, t := range triplets {
		fqdn, zone, relName := buildRecordNames(t.Domain)

		h.Logger.Info("creating TXT record",
			"domain", t.Domain,
			"fqdn", fqdn,
			"zone", zone,
			"record_name", relName,
			"token", t.TokenValue,
		)

		records := []libdns.Record{
			libdns.TXT{
				Name: relName,
				Text: t.TokenValue,
			},
		}

		_, err := h.Provider.AppendRecords(ctx, zone, records)
		if err != nil {
			return fmt.Errorf("deploy_challenge: failed to create record for %s: %w", t.Domain, err)
		}

		challengeRecords = append(challengeRecords, dns.ChallengeRecord{
			FQDN:  fqdn,
			Token: t.TokenValue,
		})
	}

	h.Logger.Info("waiting for DNS propagation", "records", len(challengeRecords))
	if err := h.Checker.WaitForPropagation(ctx, challengeRecords); err != nil {
		return fmt.Errorf("deploy_challenge: propagation failed: %w", err)
	}

	return nil
}

func (h *Handler) cleanChallenge(ctx context.Context, args []string) error {
	triplets, err := parseTriplets(args)
	if err != nil {
		return fmt.Errorf("clean_challenge: %w", err)
	}

	for _, t := range triplets {
		fqdn, zone, relName := buildRecordNames(t.Domain)

		h.Logger.Info("deleting TXT record",
			"domain", t.Domain,
			"fqdn", fqdn,
			"zone", zone,
			"record_name", relName,
		)

		records := []libdns.Record{
			libdns.TXT{
				Name: relName,
				Text: t.TokenValue,
			},
		}

		_, err := h.Provider.DeleteRecords(ctx, zone, records)
		if err != nil {
			return fmt.Errorf("clean_challenge: failed to delete record for %s: %w", t.Domain, err)
		}
	}

	h.Logger.Info("challenge records cleaned")
	return nil
}
