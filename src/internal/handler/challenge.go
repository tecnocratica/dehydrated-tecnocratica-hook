package handler

import (
	"fmt"
	"strings"
)

// Triplet represents a single dehydrated challenge triplet: domain, token filename, token value.
type Triplet struct {
	Domain        string
	TokenFilename string
	TokenValue    string
}

// parseTriplets groups the argument list into triplets of (domain, token_filename, token_value).
// dehydrated passes these as consecutive positional arguments after the hook name.
func parseTriplets(args []string) ([]Triplet, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no challenge arguments provided")
	}
	if len(args)%3 != 0 {
		return nil, fmt.Errorf("arguments must be triplets (domain, token_filename, token_value), got %d args", len(args))
	}

	triplets := make([]Triplet, 0, len(args)/3)
	for i := 0; i < len(args); i += 3 {
		triplets = append(triplets, Triplet{
			Domain:        args[i],
			TokenFilename: args[i+1],
			TokenValue:    args[i+2],
		})
	}
	return triplets, nil
}

// buildRecordNames computes the DNS names needed for an ACME challenge record.
// Returns the FQDN (e.g. "_acme-challenge.example.com."), the zone
// (e.g. "example.com."), and the record name relative to the zone
// (e.g. "_acme-challenge").
func buildRecordNames(domain string) (fqdn, zone, relName string) {
	// Strip wildcard prefix and trailing dot for a clean base domain.
	d := strings.TrimPrefix(domain, "*.")
	d = strings.TrimSuffix(d, ".")

	fqdn = "_acme-challenge." + d + "."

	// Extract zone: last two labels.
	parts := strings.Split(d, ".")
	if len(parts) < 2 {
		zone = d + "."
	} else {
		zone = strings.Join(parts[len(parts)-2:], ".") + "."
	}

	// Relative name: strip zone from FQDN.
	rel := strings.TrimSuffix(
		strings.TrimSuffix(fqdn, "."),
		"."+strings.TrimSuffix(zone, "."),
	)
	if rel == strings.TrimSuffix(zone, ".") {
		relName = "@"
	} else {
		relName = rel
	}
	return fqdn, zone, relName
}
