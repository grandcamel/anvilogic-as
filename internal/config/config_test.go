package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeKeyring struct {
	data map[string]string // account -> secret
	err  error
}

func (f fakeKeyring) Get(service, user string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if service != KeyringService {
		return "", errors.New("wrong service")
	}
	v, ok := f.data[user]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f fakeKeyring) Set(service, user, secret string) error {
	if f.err != nil {
		return f.err
	}
	f.data[user] = secret
	return nil
}

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func writeJSON(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveDefaults(t *testing.T) {
	res := Resolve(Options{
		Getenv:  envMap(nil),
		Keyring: fakeKeyring{err: errors.New("no keychain")},
		WorkDir: t.TempDir(),
		HomeDir: t.TempDir(),
	})
	if res.Profile != "default" || res.Sources["profile"] != "default" {
		t.Errorf("profile = %q (%s), want default/default", res.Profile, res.Sources["profile"])
	}
	if res.APIKey != "" || res.Sources["api_key"] != "none" {
		t.Errorf("api_key = %q (%s), want empty/none", res.APIKey, res.Sources["api_key"])
	}
	if res.AuthScheme != "Bearer" {
		t.Errorf("auth_scheme = %q, want Bearer", res.AuthScheme)
	}
	if res.BaseURL != "" {
		t.Errorf("base_url = %q, want empty (registry default applies later)", res.BaseURL)
	}
}

func TestResolveEnvBeatsKeyringAndFile(t *testing.T) {
	home := t.TempDir()
	writeJSON(t, Path(home), `{"profiles":{"default":{"api_key":"file-key","base_url":"https://file.example"}}}`)
	res := Resolve(Options{
		Getenv: envMap(map[string]string{
			EnvAPIKey:  "env-key",
			EnvBaseURL: "https://env.example",
		}),
		Keyring: fakeKeyring{data: map[string]string{"default": "keyring-key"}},
		WorkDir: t.TempDir(),
		HomeDir: home,
	})
	if res.APIKey != "env-key" || res.Sources["api_key"] != "env" {
		t.Errorf("api_key = %q (%s), want env-key/env", res.APIKey, res.Sources["api_key"])
	}
	if res.BaseURL != "https://env.example" || res.Sources["base_url"] != "env" {
		t.Errorf("base_url = %q (%s), want env", res.BaseURL, res.Sources["base_url"])
	}
}

func TestResolveKeyringBeatsConfigFile(t *testing.T) {
	home := t.TempDir()
	writeJSON(t, Path(home), `{"profiles":{"default":{"api_key":"file-key"}}}`)
	res := Resolve(Options{
		Getenv:  envMap(nil),
		Keyring: fakeKeyring{data: map[string]string{"default": "keyring-key"}},
		WorkDir: t.TempDir(),
		HomeDir: home,
	})
	if res.APIKey != "keyring-key" || res.Sources["api_key"] != "keyring" {
		t.Errorf("api_key = %q (%s), want keyring-key/keyring", res.APIKey, res.Sources["api_key"])
	}
}

func TestResolveConfigFileFallback(t *testing.T) {
	home := t.TempDir()
	writeJSON(t, Path(home), `{"profile":"prod","profiles":{"prod":{"api_key":"file-key","base_url":"https://file.example","auth_scheme":"ApiKey"}}}`)
	res := Resolve(Options{
		Getenv:  envMap(nil),
		Keyring: fakeKeyring{err: errors.New("no keychain")},
		WorkDir: t.TempDir(),
		HomeDir: home,
	})
	if res.Profile != "prod" || res.Sources["profile"] != "config-file" {
		t.Errorf("profile = %q (%s), want prod/config-file", res.Profile, res.Sources["profile"])
	}
	if res.APIKey != "file-key" || res.Sources["api_key"] != "config-file" {
		t.Errorf("api_key = %q (%s)", res.APIKey, res.Sources["api_key"])
	}
	if res.AuthScheme != "ApiKey" {
		t.Errorf("auth_scheme = %q, want ApiKey", res.AuthScheme)
	}
}

func TestResolveClaudeSettingsCascade(t *testing.T) {
	work := t.TempDir()
	home := t.TempDir()
	writeJSON(t, Path(home), `{"profiles":{"local-prof":{"base_url":"https://file.example"}}}`)
	writeJSON(t, filepath.Join(work, ".claude", "settings.json"), `{"anvilogic":{"base_url":"https://shared.example","profile":"shared-prof"}}`)

	// settings.json beats config file.
	res := Resolve(Options{
		Getenv:  envMap(nil),
		Keyring: fakeKeyring{err: errors.New("no keychain")},
		WorkDir: work,
		HomeDir: home,
	})
	if res.BaseURL != "https://shared.example" || res.Sources["base_url"] != "claude-settings" {
		t.Errorf("base_url = %q (%s), want shared/claude-settings", res.BaseURL, res.Sources["base_url"])
	}
	if res.Profile != "shared-prof" {
		t.Errorf("profile = %q, want shared-prof", res.Profile)
	}

	// settings.local.json beats settings.json.
	writeJSON(t, filepath.Join(work, ".claude", "settings.local.json"), `{"anvilogic":{"base_url":"https://local.example","profile":"local-prof"}}`)
	res = Resolve(Options{
		Getenv:  envMap(nil),
		Keyring: fakeKeyring{err: errors.New("no keychain")},
		WorkDir: work,
		HomeDir: home,
	})
	if res.BaseURL != "https://local.example" || res.Sources["base_url"] != "claude-settings-local" {
		t.Errorf("base_url = %q (%s), want local/claude-settings-local", res.BaseURL, res.Sources["base_url"])
	}
	if res.Profile != "local-prof" {
		t.Errorf("profile = %q, want local-prof", res.Profile)
	}

	// env beats .claude settings.
	res = Resolve(Options{
		Getenv:  envMap(map[string]string{EnvProfile: "env-prof", EnvBaseURL: "https://env.example"}),
		Keyring: fakeKeyring{err: errors.New("no keychain")},
		WorkDir: work,
		HomeDir: home,
	})
	if res.Profile != "env-prof" || res.BaseURL != "https://env.example" {
		t.Errorf("env should win: profile=%q base_url=%q", res.Profile, res.BaseURL)
	}

	// flag beats env.
	res = Resolve(Options{
		ProfileFlag: "flag-prof",
		BaseURLFlag: "https://flag.example",
		Getenv:      envMap(map[string]string{EnvProfile: "env-prof", EnvBaseURL: "https://env.example"}),
		Keyring:     fakeKeyring{err: errors.New("no keychain")},
		WorkDir:     work,
		HomeDir:     home,
	})
	if res.Profile != "flag-prof" || res.BaseURL != "https://flag.example" {
		t.Errorf("flag should win: profile=%q base_url=%q", res.Profile, res.BaseURL)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	home := t.TempDir()
	if err := SetProfileKey(home, "default", "api_key", "secret"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(Path(home))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file mode = %o, want 600", perm)
	}
	f, err := LoadFile(home)
	if err != nil {
		t.Fatal(err)
	}
	if f.Profiles["default"].APIKey != "secret" {
		t.Errorf("round-trip api_key = %q", f.Profiles["default"].APIKey)
	}
}

func TestMask(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"short":            "****",
		"abcdefgh":         "****",
		"anvk-1234567890x": "anvk...890x",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}
