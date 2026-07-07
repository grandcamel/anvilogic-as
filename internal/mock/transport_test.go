package mock

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grandcamel/anvilogic-as/internal/registry"
)

func fixturesDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "fixtures")
}

func doReq(t *testing.T, tr *Transport, method, path string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, "https://secure.anvilogic.com"+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("mock transport must not error: %v", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp, string(b)
}

func TestFixtureName(t *testing.T) {
	if got := FixtureName("get", "/v1/ping"); got != "GET_v1_ping.json" {
		t.Errorf("FixtureName = %q", got)
	}
	if got := FixtureName("POST", "/v1/a/b/"); got != "POST_v1_a_b.json" {
		t.Errorf("FixtureName = %q", got)
	}
}

func TestConventionalFixtureMatch(t *testing.T) {
	tr := NewTransport(nil, fixturesDir(t))
	resp, body := doReq(t, tr, http.MethodGet, "/v1/ping")
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(body, `"anvilogic-as mock"`) {
		t.Errorf("body = %s", body)
	}
}

func TestRegistryFixtureKeyMatch(t *testing.T) {
	reg, err := registry.Parse([]byte(`
version: 1
base_url: https://x
endpoints:
  ping.alias:
    domain: platform
    method: GET
    path: /alias/ping
    summary: s
    pagination: {style: none}
    auth: x
    risk: read
    confidence: confirmed
    fixture: GET_v1_ping.json
`))
	if err != nil {
		t.Fatal(err)
	}
	tr := NewTransport(reg, fixturesDir(t))
	resp, body := doReq(t, tr, http.MethodGet, "/alias/ping")
	if resp.StatusCode != 200 || !strings.Contains(body, "anvilogic-as mock") {
		t.Errorf("registry fixture match failed: %d %s", resp.StatusCode, body)
	}
}

func TestUnmatchedIs404WithHint(t *testing.T) {
	tr := NewTransport(nil, fixturesDir(t))
	resp, body := doReq(t, tr, http.MethodGet, "/v1/nonexistent")
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if !strings.Contains(body, "add fixture or registry entry") {
		t.Errorf("hint missing from body: %s", body)
	}
}

func TestEnabledAndFixturesDirEnv(t *testing.T) {
	t.Setenv(EnvMockMode, "1")
	if !Enabled() {
		t.Error("Enabled() should be true")
	}
	t.Setenv(EnvMockMode, "0")
	if Enabled() {
		t.Error("Enabled() should be false")
	}

	t.Setenv(EnvFixturesDir, "/custom/dir")
	if got := FixturesDir(); got != "/custom/dir" {
		t.Errorf("FixturesDir = %q", got)
	}
	if err := os.Unsetenv(EnvFixturesDir); err != nil {
		t.Fatal(err)
	}
	if got := FixturesDir(); got != filepath.Join("testdata", "fixtures") {
		t.Errorf("default FixturesDir = %q", got)
	}
}
