package registry

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadEmbedded parses the REAL embedded api-surface/endpoints.yaml.
func TestLoadEmbedded(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if reg.Version != 1 {
		t.Errorf("version = %d, want 1", reg.Version)
	}
	if reg.BaseURL != "https://secure.anvilogic.com" {
		t.Errorf("base_url = %q", reg.BaseURL)
	}
	if len(reg.Endpoints) < 30 {
		t.Errorf("endpoints = %d, want >= 30", len(reg.Endpoints))
	}
	if got := reg.IDs("eoi"); len(got) != 8 {
		t.Errorf("eoi ids = %v (%d), want 8", got, len(got))
	}

	deploy, ok := reg.Endpoints["detections.deploy"]
	if !ok {
		t.Fatal("detections.deploy missing")
	}
	if deploy.ID != "detections.deploy" {
		t.Errorf("ID = %q, want detections.deploy", deploy.ID)
	}
	if len(deploy.Evidence) != 2 {
		t.Errorf("detections.deploy evidence = %d, want 2", len(deploy.Evidence))
	}
	if deploy.Risk != "write" || deploy.Confidence != "inferred" {
		t.Errorf("risk/confidence = %s/%s", deploy.Risk, deploy.Confidence)
	}
	if deploy.HasKnownPath() {
		t.Error("Wave-1 entry must not report a known path")
	}
	if deploy.LastVerified != nil {
		t.Errorf("last_verified = %v, want nil", *deploy.LastVerified)
	}
}

func TestValidateEmbedded(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if errs := reg.Validate(); len(errs) != 0 {
		t.Errorf("embedded registry must validate, got: %v", errs)
	}
}

func TestValidateCatchesBadEnums(t *testing.T) {
	reg, err := Parse([]byte(`
version: 1
base_url: https://x
endpoints:
  a.b:
    domain: a
    method: TELEPORT
    path: /x
    summary: s
    pagination: {style: quantum}
    auth: bearer-api-key
    risk: reckless
    confidence: vibes
`))
	if err != nil {
		t.Fatal(err)
	}
	errs := reg.Validate()
	if len(errs) != 4 {
		t.Errorf("got %d validation errors (%v), want 4 (method, style, risk, confidence)", len(errs), errs)
	}
}

func TestParseLenientUnknownFieldsAndFixture(t *testing.T) {
	reg, err := Parse([]byte(`
version: 1
base_url: https://x
future_top_level: ignored
auth:
  probe: a.b
endpoints:
  a.b:
    domain: a
    method: GET
    path: /v1/ping
    summary: s
    pagination: {style: none}
    auth: bearer-api-key
    risk: read
    confidence: confirmed
    fixture: GET_v1_ping.json
    future_field: ignored
`))
	if err != nil {
		t.Fatalf("lenient parse failed: %v", err)
	}
	e := reg.Endpoints["a.b"]
	if e.Fixture != "GET_v1_ping.json" {
		t.Errorf("fixture = %q", e.Fixture)
	}
	if !e.HasKnownPath() {
		t.Error("concrete path should be known")
	}
	if reg.Auth == nil || reg.Auth.Probe != "a.b" {
		t.Errorf("auth.probe not parsed: %+v", reg.Auth)
	}
	if errs := reg.Validate(); len(errs) != 0 {
		t.Errorf("validate: %v", errs)
	}
}

func TestLoadOverridePathAndEnv(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "reg.yaml")
	content := []byte("version: 1\nbase_url: https://override\nendpoints:\n  a.b:\n    domain: a\n    method: GET\n    path: /p\n    summary: s\n    pagination: {style: none}\n    auth: x\n    risk: read\n    confidence: confirmed\n")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}

	reg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if reg.BaseURL != "https://override" {
		t.Errorf("flag override not applied: %q", reg.BaseURL)
	}

	t.Setenv(EnvRegistryPath, p)
	reg, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if reg.BaseURL != "https://override" {
		t.Errorf("env override not applied: %q", reg.BaseURL)
	}

	if _, err := Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("missing override file must error")
	}
}
