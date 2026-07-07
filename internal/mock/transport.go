// Package mock provides the ANVILOGIC_MOCK_MODE=1 transport: an
// http.RoundTripper that answers requests from registry entries with
// concrete paths and from JSON fixture files, never touching the network.
package mock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grandcamel/anvilogic-as/internal/registry"
)

// Environment switches.
const (
	EnvMockMode    = "ANVILOGIC_MOCK_MODE"
	EnvFixturesDir = "ANVILOGIC_FIXTURES_DIR"
)

// Enabled reports whether mock mode is on.
func Enabled() bool {
	return os.Getenv(EnvMockMode) == "1"
}

// FixturesDir returns the fixture directory: $ANVILOGIC_FIXTURES_DIR or
// testdata/fixtures relative to the working directory.
func FixturesDir() string {
	if v := os.Getenv(EnvFixturesDir); v != "" {
		return v
	}
	return filepath.Join("testdata", "fixtures")
}

// Transport is the mock RoundTripper.
type Transport struct {
	Registry *registry.Registry
	Dir      string
}

// NewTransport builds a Transport over a registry and fixture directory.
func NewTransport(reg *registry.Registry, dir string) *Transport {
	return &Transport{Registry: reg, Dir: dir}
}

// FixtureName maps METHOD + path to the conventional fixture filename,
// e.g. GET /v1/ping -> GET_v1_ping.json.
func FixtureName(method, path string) string {
	p := strings.Trim(path, "/")
	p = strings.ReplaceAll(p, "/", "_")
	return strings.ToUpper(method) + "_" + p + ".json"
}

// RoundTrip matches method+path first against registry entries with a
// concrete path (serving their fixture file when named), then against the
// conventional fixture filename. Unmatched requests get a 404-style
// response so the CLI exits 4 with a hint.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Registry entries with a concrete path.
	if t.Registry != nil {
		for _, id := range t.Registry.IDs("") {
			e := t.Registry.Endpoints[id]
			if !e.HasKnownPath() || e.Path != req.URL.Path {
				continue
			}
			if !strings.EqualFold(e.Method, req.Method) && !strings.EqualFold(e.Method, "UNKNOWN") {
				continue
			}
			if e.Fixture != "" {
				if b, err := os.ReadFile(filepath.Join(t.Dir, e.Fixture)); err == nil {
					return jsonResponse(req, http.StatusOK, b), nil
				}
			}
			break // matched an entry; fall through to filename convention
		}
	}

	// 2. Conventional fixture file.
	name := FixtureName(req.Method, req.URL.Path)
	if b, err := os.ReadFile(filepath.Join(t.Dir, name)); err == nil {
		return jsonResponse(req, http.StatusOK, b), nil
	}

	// 3. Unmatched: 404 with a hint.
	body := fmt.Sprintf(
		`{"error":{"code":"not-found","status":404,"message":"mock: no match for %s %s","hint":"add fixture or registry entry (expected %s)"}}`,
		req.Method, req.URL.Path, name)
	return jsonResponse(req, http.StatusNotFound, []byte(body)), nil
}

func jsonResponse(req *http.Request, status int, body []byte) *http.Response {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        h,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}
