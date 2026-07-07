// Package auth abstracts credential access behind a TokenSource so command
// code (and the agent driving it) never touches token mechanics directly.
package auth

import (
	"context"
	"errors"

	"github.com/grandcamel/anvilogic-as/internal/config"
)

// ErrNoToken is returned when no API key is configured anywhere in the
// config cascade.
var ErrNoToken = errors.New("no API key configured")

// TokenSource supplies a bearer credential.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Static is a TokenSource for a fixed key (from the config cascade).
type Static struct {
	key string
}

// NewStatic wraps a literal key.
func NewStatic(key string) *Static { return &Static{key: key} }

// Token returns the static key, or ErrNoToken when empty.
func (s *Static) Token(_ context.Context) (string, error) {
	if s.key == "" {
		return "", ErrNoToken
	}
	return s.key, nil
}

// FromConfig builds a TokenSource from a resolved config.
func FromConfig(res *config.Resolved) TokenSource {
	return NewStatic(res.APIKey)
}

// Header renders the Authorization header value for a token using the
// configured scheme (default "Bearer"; NOT yet publicly confirmed for
// Anvilogic — see api-surface/auth.md, hence configurable via the
// auth_scheme config key).
func Header(scheme, token string) string {
	if scheme == "" {
		scheme = config.DefaultAuthScheme
	}
	return scheme + " " + token
}
