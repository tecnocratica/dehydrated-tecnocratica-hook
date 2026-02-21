package handler

import "testing"

func TestParseTriplets(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{
			name:    "single triplet",
			args:    []string{"example.com", "token-file", "token-value"},
			want:    1,
			wantErr: false,
		},
		{
			name:    "two triplets (HOOK_CHAIN)",
			args:    []string{"a.com", "tf1", "tv1", "b.com", "tf2", "tv2"},
			want:    2,
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			want:    0,
			wantErr: true,
		},
		{
			name:    "incomplete triplet",
			args:    []string{"example.com", "token-file"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "extra arg",
			args:    []string{"example.com", "token-file", "token-value", "extra"},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triplets, err := parseTriplets(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTriplets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(triplets) != tt.want {
				t.Errorf("parseTriplets() returned %d triplets, want %d", len(triplets), tt.want)
			}
		})
	}
}

func TestParseTriplets_Values(t *testing.T) {
	triplets, err := parseTriplets([]string{"example.com", "token-file", "secret-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triplets[0].Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", triplets[0].Domain, "example.com")
	}
	if triplets[0].TokenFilename != "token-file" {
		t.Errorf("TokenFilename = %q, want %q", triplets[0].TokenFilename, "token-file")
	}
	if triplets[0].TokenValue != "secret-token" {
		t.Errorf("TokenValue = %q, want %q", triplets[0].TokenValue, "secret-token")
	}
}

func TestBuildRecordNames(t *testing.T) {
	tests := []struct {
		domain      string
		wantFQDN    string
		wantZone    string
		wantRelName string
	}{
		{"example.com", "_acme-challenge.example.com.", "example.com.", "_acme-challenge"},
		{"www.example.com", "_acme-challenge.www.example.com.", "example.com.", "_acme-challenge.www"},
		{"*.example.com", "_acme-challenge.example.com.", "example.com.", "_acme-challenge"},
		{"*.sub.example.com", "_acme-challenge.sub.example.com.", "example.com.", "_acme-challenge.sub"},
		{"example.com.", "_acme-challenge.example.com.", "example.com.", "_acme-challenge"},
		{"sub.www.example.com", "_acme-challenge.sub.www.example.com.", "example.com.", "_acme-challenge.sub.www"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			fqdn, zone, relName := buildRecordNames(tt.domain)
			if fqdn != tt.wantFQDN {
				t.Errorf("fqdn = %q, want %q", fqdn, tt.wantFQDN)
			}
			if zone != tt.wantZone {
				t.Errorf("zone = %q, want %q", zone, tt.wantZone)
			}
			if relName != tt.wantRelName {
				t.Errorf("relName = %q, want %q", relName, tt.wantRelName)
			}
		})
	}
}
