package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/grandcamel/anvilogic-as/internal/config"
	"github.com/grandcamel/anvilogic-as/internal/mock"
	"github.com/grandcamel/anvilogic-as/internal/output"
)

func runCLI(t *testing.T, version string, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd(version)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), err
}

func mockEnv(t *testing.T) {
	t.Helper()
	t.Setenv(mock.EnvMockMode, "1")
	abs, err := filepath.Abs(filepath.Join("..", "..", "testdata", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(mock.EnvFixturesDir, abs)
	// Short-circuit the config cascade before it can touch a real keychain.
	t.Setenv(config.EnvAPIKey, "test-key")
}

func exitOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var oe *output.Error
	if !errors.As(err, &oe) {
		t.Fatalf("error is not *output.Error: %v", err)
	}
	return oe.Exit
}

func TestAPIMockPing(t *testing.T) {
	mockEnv(t)
	stdout, err := runCLI(t, "test", "api", "GET", "/v1/ping")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &v); jerr != nil {
		t.Fatalf("stdout not JSON: %v (%s)", jerr, stdout)
	}
	if v.Status != "ok" || v.Message != "anvilogic-as mock" {
		t.Errorf("payload = %+v", v)
	}
}

func TestAPIMockUnmatchedExits4(t *testing.T) {
	mockEnv(t)
	_, err := runCLI(t, "test", "api", "GET", "/v1/nonexistent")
	if got := exitOf(t, err); got != output.ExitNotFound {
		t.Fatalf("exit = %d, want 4", got)
	}
	var oe *output.Error
	_ = errors.As(err, &oe)
	if oe.Hint == "" || oe.Status != 404 {
		t.Errorf("hint/status = %q/%d", oe.Hint, oe.Status)
	}
}

func TestAPIUnknownPathRegistryIDExits1WithWaveHint(t *testing.T) {
	mockEnv(t)
	_, err := runCLI(t, "test", "api", "GET", "detections.list")
	if got := exitOf(t, err); got != output.ExitValidation {
		t.Fatalf("exit = %d, want 1", got)
	}
	var oe *output.Error
	_ = errors.As(err, &oe)
	if oe.Hint != "endpoint path not yet captured (Wave 3); pass an explicit path" {
		t.Errorf("hint = %q", oe.Hint)
	}
}

func TestAPIPaginateRawPathRequiresStyle(t *testing.T) {
	mockEnv(t)
	_, err := runCLI(t, "test", "api", "GET", "/v1/items", "--paginate")
	if got := exitOf(t, err); got != output.ExitValidation {
		t.Fatalf("exit = %d, want 1", got)
	}
	var oe *output.Error
	_ = errors.As(err, &oe)
	if oe.Hint == "" {
		t.Error("expected a hint about --paginate-style")
	}
}

func TestAPIPaginateNoneStyleWorks(t *testing.T) {
	mockEnv(t)
	stdout, err := runCLI(t, "test", "--output", "ndjson", "api", "GET", "/v1/items", "--paginate", "--paginate-style", "none")
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace([]byte(stdout)), []byte("\n"))
	if len(lines) != 2 {
		t.Errorf("ndjson item lines = %d, want 2 (%s)", len(lines), stdout)
	}
}

func TestRegistryListEOIDomain(t *testing.T) {
	stdout, err := runCLI(t, "test", "registry", "list", "--domain", "eoi")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		ID     string `json:"id"`
		Domain string `json:"domain"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &rows); jerr != nil {
		t.Fatalf("stdout not JSON array: %v", jerr)
	}
	if len(rows) != 8 {
		t.Fatalf("eoi rows = %d, want 8", len(rows))
	}
	for _, r := range rows {
		if r.Domain != "eoi" {
			t.Errorf("row %s has domain %s", r.ID, r.Domain)
		}
	}
}

func TestRegistryValidateEmbedded(t *testing.T) {
	stdout, err := runCLI(t, "test", "registry", "validate")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Valid     bool `json:"valid"`
		Endpoints int  `json:"endpoints"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &v); jerr != nil {
		t.Fatal(jerr)
	}
	if !v.Valid || v.Endpoints < 30 {
		t.Errorf("validate = %+v", v)
	}
}

func TestAuthValidateNoKeyExits2(t *testing.T) {
	t.Setenv(config.EnvAPIKey, "")
	t.Setenv("HOME", t.TempDir()) // isolate from any real config file / keychain fallback
	_, err := runCLI(t, "test", "auth", "validate")
	if got := exitOf(t, err); got != output.ExitAuth {
		t.Fatalf("exit = %d, want 2", got)
	}
}

func TestVersionCheckMin(t *testing.T) {
	if _, err := runCLI(t, "1.2.3", "version", "--check-min", "1.2.0"); err != nil {
		t.Errorf("1.2.3 >= 1.2.0 should pass: %v", err)
	}
	_, err := runCLI(t, "1.2.3", "version", "--check-min", "2.0.0")
	if got := exitOf(t, err); got != output.ExitValidation {
		t.Errorf("1.2.3 < 2.0.0 should exit 1, got %d", got)
	}
	if _, err := runCLI(t, "dev", "version", "--check-min", "9.9.9"); err != nil {
		t.Errorf("dev build should pass any check-min: %v", err)
	}
}

func TestInvalidOutputMode(t *testing.T) {
	_, err := runCLI(t, "test", "--output", "yaml", "registry", "validate")
	if got := exitOf(t, err); got != output.ExitValidation {
		t.Errorf("invalid --output should exit 1, got %d", got)
	}
}
