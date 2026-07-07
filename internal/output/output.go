// Package output owns the CLI's data/diagnostic contract:
// stdout carries data only; stderr carries diagnostics and one structured
// error JSON object; the process exit code encodes the failure class.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Exit codes.
const (
	ExitOK         = 0
	ExitValidation = 1
	ExitAuth       = 2
	ExitPermission = 3
	ExitNotFound   = 4
	ExitRateLimit  = 5
	ExitConflict   = 6
	ExitServer     = 7
)

// Output modes.
const (
	ModeJSON   = "json"
	ModeNDJSON = "ndjson"
	ModeText   = "text"
)

// ValidMode reports whether m is a supported --output value.
func ValidMode(m string) bool {
	return m == ModeJSON || m == ModeNDJSON || m == ModeText
}

// Error is the CLI's structured error. It is rendered to stderr as
// {"error":{"code":...,"status":...,"hint":...}} and carries the process
// exit code.
type Error struct {
	Code    string `json:"code"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Hint    string `json:"hint,omitempty"`
	Exit    int    `json:"-"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// Validation builds an exit-1 validation error.
func Validation(msg, hint string) *Error {
	return &Error{Code: "validation", Message: msg, Hint: hint, Exit: ExitValidation}
}

// CodeForStatus maps an HTTP status to the CLI error code and exit code.
func CodeForStatus(status int) (code string, exit int) {
	switch {
	case status == 401:
		return "auth", ExitAuth
	case status == 403:
		return "permission", ExitPermission
	case status == 404:
		return "not-found", ExitNotFound
	case status == 429:
		return "rate-limit-exhausted", ExitRateLimit
	case status == 409:
		return "conflict", ExitConflict
	case status >= 500:
		return "server", ExitServer
	default:
		return "validation", ExitValidation
	}
}

// FromResponse builds an Error from an HTTP error response. If the body is
// itself a structured {"error":{...}} object, its hint/message are reused.
func FromResponse(status int, body []byte) *Error {
	code, exit := CodeForStatus(status)
	e := &Error{Code: code, Status: status, Exit: exit}
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Hint    string `json:"hint"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil {
		e.Message = payload.Error.Message
		e.Hint = payload.Error.Hint
	}
	if e.Message == "" {
		e.Message = fmt.Sprintf("HTTP %d", status)
	}
	return e
}

// WriteError renders err as structured error JSON on w (stderr by
// convention). Non-*Error values are wrapped as validation errors.
func WriteError(w io.Writer, err error) {
	e, ok := err.(*Error)
	if !ok {
		e = &Error{Code: "validation", Message: err.Error(), Exit: ExitValidation}
	}
	wrapper := struct {
		Error *Error `json:"error"`
	}{e}
	b, merr := json.Marshal(wrapper)
	if merr != nil {
		_, _ = fmt.Fprintf(w, `{"error":{"code":"validation","message":%q}}`+"\n", err.Error())
		return
	}
	_, _ = fmt.Fprintln(w, string(b))
}

// PrintData writes a raw response body according to the output mode.
// json: pretty-printed; ndjson: one JSON object per line (arrays and
// {"items":[...]}-style envelopes are split); text: the body as-is.
func PrintData(w io.Writer, mode string, body []byte) error {
	switch mode {
	case ModeNDJSON:
		return PrintNDJSON(w, splitItems(body))
	case ModeText:
		_, err := fmt.Fprintln(w, string(bytes.TrimRight(body, "\n")))
		return err
	default:
		return PrintJSON(w, body)
	}
}

// PrintJSON pretty-prints body if it is valid JSON, else writes it raw.
func PrintJSON(w io.Writer, body []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, bytes.TrimSpace(body), "", "  "); err != nil {
		_, werr := fmt.Fprintln(w, string(body))
		return werr
	}
	_, err := fmt.Fprintln(w, buf.String())
	return err
}

// PrintNDJSON writes one compact JSON value per line.
func PrintNDJSON(w io.Writer, items []json.RawMessage) error {
	for _, it := range items {
		var buf bytes.Buffer
		if err := json.Compact(&buf, it); err != nil {
			buf.Reset()
			buf.Write(it)
		}
		if _, err := fmt.Fprintln(w, buf.String()); err != nil {
			return err
		}
	}
	return nil
}

// PrintItems writes a slice of items in the requested mode: json renders a
// pretty array, ndjson one item per line, text one compact item per line.
func PrintItems(w io.Writer, mode string, items []json.RawMessage) error {
	switch mode {
	case ModeNDJSON, ModeText:
		return PrintNDJSON(w, items)
	default:
		b, err := json.Marshal(items)
		if err != nil {
			return err
		}
		return PrintJSON(w, b)
	}
}

// PrintValue marshals v and writes it in the requested mode.
func PrintValue(w io.Writer, mode string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return PrintData(w, mode, b)
}

// splitItems breaks a body into NDJSON-able items: a top-level array is
// split per element; an object with an items/data/results array is split;
// anything else is a single item.
func splitItems(body []byte) []json.RawMessage {
	trimmed := bytes.TrimSpace(body)
	var arr []json.RawMessage
	if err := json.Unmarshal(trimmed, &arr); err == nil && len(trimmed) > 0 && trimmed[0] == '[' {
		return arr
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &obj); err == nil {
		for _, key := range []string{"items", "data", "results"} {
			if raw, ok := obj[key]; ok {
				var inner []json.RawMessage
				if err := json.Unmarshal(raw, &inner); err == nil {
					return inner
				}
			}
		}
	}
	return []json.RawMessage{json.RawMessage(trimmed)}
}
