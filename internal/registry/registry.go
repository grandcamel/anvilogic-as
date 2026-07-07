// Package registry parses and validates the api-surface endpoint registry
// (api-surface/endpoints.yaml). Parsing is lenient: unknown YAML fields are
// ignored so future waves can extend the schema without breaking old CLIs.
package registry

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	anvilogicas "github.com/grandcamel/anvilogic-as"
)

// EnvRegistryPath overrides the embedded registry when set.
const EnvRegistryPath = "ANVILOGIC_REGISTRY_PATH"

// Registry mirrors the endpoints.yaml schema.
type Registry struct {
	Version           int                 `yaml:"version" json:"version"`
	BaseURL           string              `yaml:"base_url" json:"base_url"`
	BaseURLConfidence string              `yaml:"base_url_confidence" json:"base_url_confidence,omitempty"`
	Generated         string              `yaml:"generated" json:"generated"`
	Auth              *AuthInfo           `yaml:"auth" json:"auth,omitempty"`
	Endpoints         map[string]Endpoint `yaml:"endpoints" json:"endpoints"`
}

// AuthInfo is optional registry-level auth metadata (e.g. a probe endpoint
// id that `auth validate` can call). Wave-1 registries do not set it.
type AuthInfo struct {
	Probe string `yaml:"probe" json:"probe,omitempty"`
}

// Endpoint is a single registry entry.
type Endpoint struct {
	ID           string     `yaml:"-" json:"id"`
	Domain       string     `yaml:"domain" json:"domain"`
	Method       string     `yaml:"method" json:"method"`
	Path         string     `yaml:"path" json:"path"`
	Summary      string     `yaml:"summary" json:"summary"`
	Params       []Param    `yaml:"params" json:"params"`
	Pagination   Pagination `yaml:"pagination" json:"pagination"`
	Auth         string     `yaml:"auth" json:"auth"`
	Risk         string     `yaml:"risk" json:"risk"`
	Confidence   string     `yaml:"confidence" json:"confidence"`
	Evidence     []Evidence `yaml:"evidence" json:"evidence"`
	FirstSeen    string     `yaml:"first_seen" json:"first_seen"`
	LastVerified *string    `yaml:"last_verified" json:"last_verified"`
	// Fixture optionally names a mock fixture file for this endpoint.
	// Wave-1 entries lack it.
	Fixture string `yaml:"fixture" json:"fixture,omitempty"`
}

// Param documents a known/inferred request parameter.
type Param struct {
	Name string `yaml:"name" json:"name"`
	Note string `yaml:"note" json:"note,omitempty"`
}

// Pagination records the pagination style of an endpoint.
type Pagination struct {
	Style string `yaml:"style" json:"style"`
}

// Evidence records a public source backing an entry.
type Evidence struct {
	Source string `yaml:"source" json:"source"`
	Kind   string `yaml:"kind" json:"kind"`
	Note   string `yaml:"note" json:"note,omitempty"`
}

// HasKnownPath reports whether the endpoint has a concrete captured path.
func (e Endpoint) HasKnownPath() bool {
	return e.Path != "" && !strings.EqualFold(e.Path, "UNKNOWN")
}

// Load returns the registry from overridePath if non-empty, else from
// $ANVILOGIC_REGISTRY_PATH, else from the embedded copy.
func Load(overridePath string) (*Registry, error) {
	path := overridePath
	if path == "" {
		path = os.Getenv(EnvRegistryPath)
	}
	data := anvilogicas.EndpointsYAML
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read registry %s: %w", path, err)
		}
		data = b
	}
	return Parse(data)
}

// Parse decodes registry YAML, ignoring unknown fields.
func Parse(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	for id, e := range reg.Endpoints {
		e.ID = id
		reg.Endpoints[id] = e
	}
	return &reg, nil
}

// IDs returns all endpoint ids, sorted; domain filters when non-empty.
func (r *Registry) IDs(domain string) []string {
	ids := make([]string, 0, len(r.Endpoints))
	for id, e := range r.Endpoints {
		if domain != "" && e.Domain != domain {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Enum values accepted by Validate.
var (
	validMethods = set("UNKNOWN", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS")
	validStyles  = set("unknown", "none", "offset_limit", "page_number", "cursor", "link_header")
	validRisks   = set("read", "write", "bulk", "destructive")
	validConfs   = set("confirmed", "inferred", "unknown")
)

func set(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// Validate checks the registry schema and enum values. It returns all
// problems found, empty when the registry is valid.
func (r *Registry) Validate() []error {
	var errs []error
	if r.Version < 1 {
		errs = append(errs, fmt.Errorf("version: must be >= 1, got %d", r.Version))
	}
	if r.BaseURL == "" {
		errs = append(errs, fmt.Errorf("base_url: required"))
	}
	if len(r.Endpoints) == 0 {
		errs = append(errs, fmt.Errorf("endpoints: at least one entry required"))
	}
	for _, id := range r.IDs("") {
		e := r.Endpoints[id]
		if e.Domain == "" {
			errs = append(errs, fmt.Errorf("%s: domain required", id))
		}
		if !validMethods[strings.ToUpper(e.Method)] {
			errs = append(errs, fmt.Errorf("%s: invalid method %q", id, e.Method))
		}
		if !validStyles[e.Pagination.Style] {
			errs = append(errs, fmt.Errorf("%s: invalid pagination.style %q", id, e.Pagination.Style))
		}
		if !validRisks[e.Risk] {
			errs = append(errs, fmt.Errorf("%s: invalid risk %q", id, e.Risk))
		}
		if !validConfs[e.Confidence] {
			errs = append(errs, fmt.Errorf("%s: invalid confidence %q", id, e.Confidence))
		}
		if e.Auth == "" {
			errs = append(errs, fmt.Errorf("%s: auth required", id))
		}
		if e.Summary == "" {
			errs = append(errs, fmt.Errorf("%s: summary required", id))
		}
	}
	return errs
}
