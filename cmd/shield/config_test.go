package main

import (
	"testing"

	"github.com/goccy/go-yaml"
)

func TestSHIELDConfigRoundTrip(t *testing.T) {
	shields := map[string]*SHIELD{
		"prod": {
			URL:                "https://shield.example.com",
			Session:            "s-abc123",
			InsecureSkipVerify: false,
			CACertificate:      "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
		"dev": {
			URL:                "http://localhost:8888",
			Session:            "s-dev456",
			InsecureSkipVerify: true,
		},
	}

	b, err := yaml.Marshal(shields)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got map[string]*SHIELD
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	for name, want := range shields {
		g, ok := got[name]
		if !ok {
			t.Errorf("missing key %q after round-trip", name)
			continue
		}
		if g.URL != want.URL {
			t.Errorf("%s URL = %q, want %q", name, g.URL, want.URL)
		}
		if g.Session != want.Session {
			t.Errorf("%s Session = %q, want %q", name, g.Session, want.Session)
		}
		if g.InsecureSkipVerify != want.InsecureSkipVerify {
			t.Errorf("%s InsecureSkipVerify = %v, want %v", name, g.InsecureSkipVerify, want.InsecureSkipVerify)
		}
		if g.CACertificate != want.CACertificate {
			t.Errorf("%s CACertificate = %q, want %q", name, g.CACertificate, want.CACertificate)
		}
	}
}

func TestLegacyConfigParsing(t *testing.T) {
	input := `
backend: prod
backends:
  https://shield.example.com: session-token-abc
aliases:
  prod: https://shield.example.com
properties:
  prod:
    skip_ssl_validation: true
    ca_cert: "my-ca-cert"
`

	var legacy LegacyConfig
	if err := yaml.Unmarshal([]byte(input), &legacy); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if legacy.Backend != "prod" {
		t.Errorf("Backend = %q, want %q", legacy.Backend, "prod")
	}
	if legacy.Aliases["prod"] != "https://shield.example.com" {
		t.Errorf("Aliases[prod] = %q, want %q", legacy.Aliases["prod"], "https://shield.example.com")
	}
	props := legacy.Properties["prod"]
	if !props.InsecureSkipVerify {
		t.Error("Properties[prod].InsecureSkipVerify = false, want true")
	}
	if props.CACert != "my-ca-cert" {
		t.Errorf("Properties[prod].CACert = %q, want %q", props.CACert, "my-ca-cert")
	}
}

func TestImportManifestParsing(t *testing.T) {
	input := `
core: https://shield.example.com
token: my-token
insecure_skip_verify: false
global:
  storage:
    - name: S3 Storage
      summary: Global S3
      agent: "10.0.0.5:5444"
      plugin: s3
      config:
        bucket: my-bucket
        prefix: backups
        port: 443
        enabled: true
tenants:
  - name: Production
    members:
      - user: admin@local
        role: admin
    systems:
      - name: My Database
        summary: Production DB
        agent: "10.0.0.6:5444"
        plugin: postgres
        config:
          host: db.example.com
          port: 5432
        jobs:
          - name: Daily
            when: daily 4am
            retain: 7d
            storage: S3 Storage
            retries: 3
`

	var m ImportManifest
	if err := yaml.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if m.Core != "https://shield.example.com" {
		t.Errorf("Core = %q, want %q", m.Core, "https://shield.example.com")
	}
	if m.Token != "my-token" {
		t.Errorf("Token = %q, want %q", m.Token, "my-token")
	}
	if len(m.Global.Storage) != 1 {
		t.Fatalf("Global.Storage length = %d, want 1", len(m.Global.Storage))
	}

	storage := m.Global.Storage[0]
	if storage.Name != "S3 Storage" {
		t.Errorf("Storage.Name = %q, want %q", storage.Name, "S3 Storage")
	}

	// Verify map[string]interface{} config with mixed types
	bucket, ok := storage.Config["bucket"]
	if !ok || bucket != "my-bucket" {
		t.Errorf("Storage.Config[bucket] = %v, want %q", bucket, "my-bucket")
	}
	enabled, ok := storage.Config["enabled"]
	if !ok || enabled != true {
		t.Errorf("Storage.Config[enabled] = %v, want true", enabled)
	}

	if len(m.Tenants) != 1 {
		t.Fatalf("Tenants length = %d, want 1", len(m.Tenants))
	}
	if m.Tenants[0].Name != "Production" {
		t.Errorf("Tenants[0].Name = %q, want %q", m.Tenants[0].Name, "Production")
	}
	if len(m.Tenants[0].Systems) != 1 {
		t.Fatalf("Systems length = %d, want 1", len(m.Tenants[0].Systems))
	}

	// Verify numeric value in map[string]interface{}
	sys := m.Tenants[0].Systems[0]
	port, ok := sys.Config["port"]
	if !ok {
		t.Fatal("Systems[0].Config[port] missing")
	}
	// YAML libraries may decode integers as int, int64, or uint64
	switch v := port.(type) {
	case int:
		if v != 5432 {
			t.Errorf("Systems[0].Config[port] = %d, want 5432", v)
		}
	case int64:
		if v != 5432 {
			t.Errorf("Systems[0].Config[port] = %d, want 5432", v)
		}
	case uint64:
		if v != 5432 {
			t.Errorf("Systems[0].Config[port] = %d, want 5432", v)
		}
	default:
		t.Errorf("Systems[0].Config[port] has unexpected type %T", port)
	}
}
