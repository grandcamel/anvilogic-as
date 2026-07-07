// Package httpx wraps net/http with the CLI's transport policy: default
// 30s timeout, context cancellation, a stable User-Agent, and exponential
// backoff with jitter on 429/5xx/transient errors honoring Retry-After.
// Non-idempotent methods (POST/PATCH) are retried ONLY on a 429 that
// carries a Retry-After header.
package httpx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// DefaultTimeout is the per-request timeout when the caller does not
// override it via --timeout.
const DefaultTimeout = 30 * time.Second

// Client is a retrying HTTP client.
type Client struct {
	HTTP        *http.Client
	UserAgent   string
	MaxAttempts int           // total attempts, default 4
	BaseBackoff time.Duration // default 500ms
	MaxBackoff  time.Duration // default 8s
	// Sleep is injectable for tests; defaults to a context-aware sleep.
	Sleep func(ctx context.Context, d time.Duration) error
}

// New builds a Client with the default policy.
func New(version string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{
		HTTP:        &http.Client{Timeout: timeout},
		UserAgent:   UserAgent(version),
		MaxAttempts: 4,
		BaseBackoff: 500 * time.Millisecond,
		MaxBackoff:  8 * time.Second,
		Sleep:       sleepCtx,
	}
}

// UserAgent renders the CLI User-Agent string.
func UserAgent(version string) string {
	return fmt.Sprintf("anvilogic-as/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
}

// Do performs the request with retries. The body (may be nil) is replayed
// on each attempt. The caller owns closing the returned response body.
func (c *Client) Do(ctx context.Context, method, url string, header http.Header, body []byte) (*http.Response, error) {
	maxAttempts := c.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 4
	}
	var lastErr error
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader(body))
		if err != nil {
			return nil, err
		}
		for k, vs := range header {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			// Transient network error: retry idempotent methods only,
			// and never after context cancellation.
			if ctx.Err() != nil || !idempotent(method) || attempt >= maxAttempts {
				return nil, err
			}
			if serr := c.Sleep(ctx, c.backoff(attempt)); serr != nil {
				return nil, lastErr
			}
			continue
		}

		retry, wait := c.retryDecision(method, resp, attempt, maxAttempts)
		if !retry {
			return resp, nil
		}
		// Drain and close before retrying so the connection is reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if serr := c.Sleep(ctx, wait); serr != nil {
			return nil, serr
		}
	}
}

func bodyReader(body []byte) io.Reader {
	if body == nil {
		return nil
	}
	return bytes.NewReader(body)
}

func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// retryDecision applies the policy: 429 retries when idempotent or when a
// Retry-After header is present; 5xx retries only for idempotent methods.
func (c *Client) retryDecision(method string, resp *http.Response, attempt, maxAttempts int) (bool, time.Duration) {
	if attempt >= maxAttempts {
		return false, 0
	}
	retryAfter, hasRetryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		if !idempotent(method) && !hasRetryAfter {
			return false, 0
		}
		if hasRetryAfter {
			return true, retryAfter
		}
		return true, c.backoff(attempt)
	case resp.StatusCode >= 500:
		if !idempotent(method) {
			return false, 0
		}
		if hasRetryAfter {
			return true, retryAfter
		}
		return true, c.backoff(attempt)
	}
	return false, 0
}

// backoff returns base*2^(attempt-1) plus up to 50% jitter, capped.
func (c *Client) backoff(attempt int) time.Duration {
	base := c.BaseBackoff
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	maxB := c.MaxBackoff
	if maxB <= 0 {
		maxB = 8 * time.Second
	}
	d := base << (attempt - 1)
	if d > maxB {
		d = maxB
	}
	jitter := time.Duration(rand.Int64N(int64(d)/2 + 1))
	if d+jitter > maxB {
		return maxB
	}
	return d + jitter
}

func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
