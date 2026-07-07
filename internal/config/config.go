// Package config resolves CLI configuration through a fixed cascade
// (first hit wins per key):
//
//	env > OS keychain > .claude/settings.local.json > .claude/settings.json
//	    > ~/.config/anvilogic-as/config.json > defaults
//
// Profiles are named sections {base_url, auth_scheme} in the config file;
// the keychain account is the profile name.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// Environment variables and defaults.
const (
	EnvAPIKey  = "ANVILOGIC_API_KEY"
	EnvBaseURL = "ANVILOGIC_BASE_URL"
	EnvProfile = "ANVILOGIC_PROFILE"

	KeyringService = "anvilogic-as"

	DefaultProfile    = "default"
	DefaultAuthScheme = "Bearer" // NOT publicly confirmed; see api-surface/auth.md
)

// Keyring abstracts the OS keychain so tests can fake it.
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
}

// SystemKeyring is the zalando/go-keyring backed implementation.
type SystemKeyring struct{}

// Get reads a secret from the OS keychain.
func (SystemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

// Set writes a secret to the OS keychain.
func (SystemKeyring) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

// File is the on-disk config file (~/.config/anvilogic-as/config.json).
type File struct {
	Profile  string             `json:"profile,omitempty"`
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// Profile is a named config section.
type Profile struct {
	BaseURL    string `json:"base_url,omitempty"`
	AuthScheme string `json:"auth_scheme,omitempty"`
	// APIKey is the file fallback used only when the OS keychain is
	// unavailable. The keychain is always preferred.
	APIKey string `json:"api_key,omitempty"`
}

// claudeSettings is the shape read from .claude/settings(.local).json.
type claudeSettings struct {
	Anvilogic struct {
		BaseURL string `json:"base_url"`
		Profile string `json:"profile"`
	} `json:"anvilogic"`
}

// Options parameterizes resolution; zero values fall back to the real
// environment (os.Getenv, OS keychain, cwd, home).
type Options struct {
	ProfileFlag string
	BaseURLFlag string
	Getenv      func(string) string
	Keyring     Keyring
	WorkDir     string
	HomeDir     string
}

// Resolved is the outcome of the cascade. Sources records, per key, which
// layer supplied the value (flag, env, keyring, claude-settings-local,
// claude-settings, config-file, default, none).
type Resolved struct {
	Profile    string
	APIKey     string
	BaseURL    string
	AuthScheme string
	Sources    map[string]string
}

func (o *Options) fill() {
	if o.Getenv == nil {
		o.Getenv = os.Getenv
	}
	if o.Keyring == nil {
		o.Keyring = SystemKeyring{}
	}
	if o.WorkDir == "" {
		o.WorkDir, _ = os.Getwd()
	}
	if o.HomeDir == "" {
		o.HomeDir, _ = os.UserHomeDir()
	}
}

// Path returns the config file path under home.
func Path(home string) string {
	return filepath.Join(home, ".config", "anvilogic-as", "config.json")
}

// Resolve runs the cascade. It never fails: missing layers are skipped.
func Resolve(opts Options) *Resolved {
	opts.fill()
	src := map[string]string{}
	res := &Resolved{Sources: src}

	local := readClaude(filepath.Join(opts.WorkDir, ".claude", "settings.local.json"))
	shared := readClaude(filepath.Join(opts.WorkDir, ".claude", "settings.json"))
	file, _ := LoadFile(opts.HomeDir)

	// Profile: flag > env > .claude local > .claude > config file > default.
	switch {
	case opts.ProfileFlag != "":
		res.Profile, src["profile"] = opts.ProfileFlag, "flag"
	case opts.Getenv(EnvProfile) != "":
		res.Profile, src["profile"] = opts.Getenv(EnvProfile), "env"
	case local != nil && local.Anvilogic.Profile != "":
		res.Profile, src["profile"] = local.Anvilogic.Profile, "claude-settings-local"
	case shared != nil && shared.Anvilogic.Profile != "":
		res.Profile, src["profile"] = shared.Anvilogic.Profile, "claude-settings"
	case file.Profile != "":
		res.Profile, src["profile"] = file.Profile, "config-file"
	default:
		res.Profile, src["profile"] = DefaultProfile, "default"
	}

	prof := file.Profiles[res.Profile]

	// API key: env > keychain > config file > none.
	if v := opts.Getenv(EnvAPIKey); v != "" {
		res.APIKey, src["api_key"] = v, "env"
	} else if v, err := opts.Keyring.Get(KeyringService, res.Profile); err == nil && v != "" {
		res.APIKey, src["api_key"] = v, "keyring"
	} else if prof.APIKey != "" {
		res.APIKey, src["api_key"] = prof.APIKey, "config-file"
	} else {
		src["api_key"] = "none"
	}

	// Base URL: flag > env > .claude local > .claude > config file > empty
	// (caller falls back to the registry base_url).
	switch {
	case opts.BaseURLFlag != "":
		res.BaseURL, src["base_url"] = opts.BaseURLFlag, "flag"
	case opts.Getenv(EnvBaseURL) != "":
		res.BaseURL, src["base_url"] = opts.Getenv(EnvBaseURL), "env"
	case local != nil && local.Anvilogic.BaseURL != "":
		res.BaseURL, src["base_url"] = local.Anvilogic.BaseURL, "claude-settings-local"
	case shared != nil && shared.Anvilogic.BaseURL != "":
		res.BaseURL, src["base_url"] = shared.Anvilogic.BaseURL, "claude-settings"
	case prof.BaseURL != "":
		res.BaseURL, src["base_url"] = prof.BaseURL, "config-file"
	default:
		src["base_url"] = "default"
	}

	// Auth scheme: config file profile > default "Bearer".
	if prof.AuthScheme != "" {
		res.AuthScheme, src["auth_scheme"] = prof.AuthScheme, "config-file"
	} else {
		res.AuthScheme, src["auth_scheme"] = DefaultAuthScheme, "default"
	}

	return res
}

// LoadFile reads the config file; a missing file yields an empty File.
func LoadFile(home string) (File, error) {
	var f File
	b, err := os.ReadFile(Path(home))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return f, nil
		}
		return f, err
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", Path(home), err)
	}
	return f, nil
}

// SaveFile writes the config file with 0600 permissions (0700 dir).
func SaveFile(home string, f File) error {
	p := Path(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o600)
}

// SetProfileKey updates one key in a named profile section of the config
// file and persists it.
func SetProfileKey(home, profile, key, value string) error {
	f, err := LoadFile(home)
	if err != nil {
		return err
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	p := f.Profiles[profile]
	switch key {
	case "base_url":
		p.BaseURL = value
	case "auth_scheme":
		p.AuthScheme = value
	case "api_key":
		p.APIKey = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	f.Profiles[profile] = p
	return SaveFile(home, f)
}

// SetDefaultProfile persists the top-level default profile name.
func SetDefaultProfile(home, profile string) error {
	f, err := LoadFile(home)
	if err != nil {
		return err
	}
	f.Profile = profile
	return SaveFile(home, f)
}

// Mask returns a masked rendering of a secret for display.
func Mask(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func readClaude(path string) *claudeSettings {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var s claudeSettings
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	return &s
}
