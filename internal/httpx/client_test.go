package httpx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testClient() (*Client, *[]time.Duration) {
	var slept []time.Duration
	c := New("test", time.Second)
	c.BaseBackoff = time.Millisecond
	c.Sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}
	return c, &slept
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, _ := testClient()
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := hits.Load(); got != 3 {
		t.Errorf("hits = %d, want 3", got)
	}
}

func TestRetryExhaustionReturnsLastResponse(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := testClient()
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if got := hits.Load(); got != 4 {
		t.Errorf("hits = %d, want 4 (MaxAttempts)", got)
	}
}

func TestRetryAfterHonored(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, slept := testClient()
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(*slept) != 1 || (*slept)[0] != 3*time.Second {
		t.Errorf("slept = %v, want [3s] (Retry-After honored)", *slept)
	}
}

func TestPostNotRetriedOn500(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, _ := testClient()
	resp, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("hits = %d, want 1 (POST must not retry on 500)", got)
	}
}

func TestPostRetriedOn429WithRetryAfterOnly(t *testing.T) {
	var withHits, withoutHits atomic.Int32

	with := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if withHits.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"a":1}` {
			t.Errorf("retried body = %q, want original body replayed", body)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer with.Close()

	without := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		withoutHits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer without.Close()

	c, _ := testClient()

	resp, err := c.Do(context.Background(), http.MethodPost, with.URL, nil, []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || withHits.Load() != 2 {
		t.Errorf("with Retry-After: status=%d hits=%d, want 200/2", resp.StatusCode, withHits.Load())
	}

	resp, err = c.Do(context.Background(), http.MethodPost, without.URL, nil, []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 429 || withoutHits.Load() != 1 {
		t.Errorf("without Retry-After: status=%d hits=%d, want 429/1", resp.StatusCode, withoutHits.Load())
	}
}

func TestTransientNetErrorRetriedForGET(t *testing.T) {
	// A server that closes immediately produces a transient error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // connection refused from now on

	c, slept := testClient()
	_, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err == nil {
		t.Fatal("want error from refused connection")
	}
	if len(*slept) != 3 {
		t.Errorf("slept %d times, want 3 (4 attempts)", len(*slept))
	}

	*slept = (*slept)[:0]
	_, err = c.Do(context.Background(), http.MethodPost, srv.URL, nil, nil)
	if err == nil {
		t.Fatal("want error from refused connection")
	}
	if len(*slept) != 0 {
		t.Errorf("POST slept %d times, want 0 (no transient retry)", len(*slept))
	}
}

func TestUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	c := New("1.2.3", time.Second)
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !strings.HasPrefix(ua, "anvilogic-as/1.2.3 (") {
		t.Errorf("User-Agent = %q", ua)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("2"); !ok || d != 2*time.Second {
		t.Errorf("seconds form: %v %v", d, ok)
	}
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	if d, ok := parseRetryAfter(future); !ok || d <= 0 || d > 31*time.Second {
		t.Errorf("date form: %v %v", d, ok)
	}
	if _, ok := parseRetryAfter(""); ok {
		t.Error("empty should not parse")
	}
	if _, ok := parseRetryAfter("bogus"); ok {
		t.Error("bogus should not parse")
	}
}
