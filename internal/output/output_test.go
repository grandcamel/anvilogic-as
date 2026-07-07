package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCodeForStatusMapping(t *testing.T) {
	cases := []struct {
		status int
		code   string
		exit   int
	}{
		{400, "validation", ExitValidation},
		{422, "validation", ExitValidation},
		{401, "auth", ExitAuth},
		{403, "permission", ExitPermission},
		{404, "not-found", ExitNotFound},
		{429, "rate-limit-exhausted", ExitRateLimit},
		{409, "conflict", ExitConflict},
		{500, "server", ExitServer},
		{503, "server", ExitServer},
	}
	for _, c := range cases {
		code, exit := CodeForStatus(c.status)
		if code != c.code || exit != c.exit {
			t.Errorf("CodeForStatus(%d) = %s/%d, want %s/%d", c.status, code, exit, c.code, c.exit)
		}
	}
}

func TestFromResponseReusesStructuredHint(t *testing.T) {
	body := []byte(`{"error":{"code":"not-found","status":404,"message":"nope","hint":"add fixture or registry entry"}}`)
	e := FromResponse(404, body)
	if e.Exit != ExitNotFound || e.Code != "not-found" {
		t.Errorf("exit/code = %d/%s", e.Exit, e.Code)
	}
	if e.Hint != "add fixture or registry entry" || e.Message != "nope" {
		t.Errorf("hint/message not propagated: %+v", e)
	}

	e = FromResponse(500, []byte("plain text"))
	if e.Exit != ExitServer || e.Message != "HTTP 500" {
		t.Errorf("plain body: %+v", e)
	}
}

func TestWriteErrorShape(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, &Error{Code: "auth", Status: 401, Hint: "h", Exit: ExitAuth})
	var payload struct {
		Error struct {
			Code   string `json:"code"`
			Status int    `json:"status"`
			Hint   string `json:"hint"`
		} `json:"error"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("stderr JSON invalid: %v (%s)", err, buf.String())
	}
	if payload.Error.Code != "auth" || payload.Error.Status != 401 || payload.Error.Hint != "h" {
		t.Errorf("payload = %+v", payload)
	}
}

func TestPrintDataModes(t *testing.T) {
	body := []byte(`{"items":[{"id":1},{"id":2}]}`)

	var buf bytes.Buffer
	if err := PrintData(&buf, ModeJSON, body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\n  \"items\"") {
		t.Errorf("json mode not pretty: %q", buf.String())
	}

	buf.Reset()
	if err := PrintData(&buf, ModeNDJSON, body); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 || lines[0] != `{"id":1}` || lines[1] != `{"id":2}` {
		t.Errorf("ndjson lines = %q", lines)
	}

	buf.Reset()
	if err := PrintData(&buf, ModeText, []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello\n" {
		t.Errorf("text mode = %q", buf.String())
	}
}

func TestValidMode(t *testing.T) {
	for _, m := range []string{"json", "ndjson", "text"} {
		if !ValidMode(m) {
			t.Errorf("%s should be valid", m)
		}
	}
	if ValidMode("yaml") {
		t.Error("yaml should be invalid")
	}
}
