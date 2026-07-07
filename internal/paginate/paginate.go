// Package paginate walks paged API responses. The style is normally
// selected from the registry entry's pagination.style; a raw path with
// --paginate requires an explicit --paginate-style.
package paginate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Canonical style names.
const (
	StyleNone        = "none"
	StyleOffsetLimit = "offset_limit"
	StylePageNumber  = "page_number"
	StyleCursor      = "cursor"
	StyleLinkHeader  = "link_header"
)

// Paginator advances through pages. Next inspects the previous request and
// its response and returns the request for the next page, the items on the
// current page, and whether iteration is complete.
type Paginator interface {
	Next(prevReq *http.Request, resp *http.Response, respBody []byte) (nextReq *http.Request, items []json.RawMessage, done bool, err error)
}

// Canonicalize maps style aliases to canonical names ("" stays "").
func Canonicalize(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "":
		return ""
	case StyleNone:
		return StyleNone
	case StyleOffsetLimit, "offsetlimit", "offset-limit", "offset":
		return StyleOffsetLimit
	case StylePageNumber, "pagenumber", "page-number", "page":
		return StylePageNumber
	case StyleCursor:
		return StyleCursor
	case StyleLinkHeader, "linkheader", "link-header", "link":
		return StyleLinkHeader
	default:
		return strings.ToLower(style)
	}
}

// ForStyle returns the paginator for a style. Unknown or empty styles are
// an error: the caller must supply --paginate-style.
func ForStyle(style string) (Paginator, error) {
	switch Canonicalize(style) {
	case StyleNone:
		return nonePaginator{}, nil
	case StyleOffsetLimit:
		return offsetLimitPaginator{}, nil
	case StylePageNumber:
		return pageNumberPaginator{}, nil
	case StyleCursor:
		return cursorPaginator{}, nil
	case StyleLinkHeader:
		return linkHeaderPaginator{}, nil
	case "", "unknown":
		return nil, fmt.Errorf("pagination style is unknown")
	default:
		return nil, fmt.Errorf("unsupported pagination style %q", style)
	}
}

// ExtractItems pulls the item list from a response body: a top-level JSON
// array, or the first of items/data/results in an object; otherwise the
// whole body is a single item.
func ExtractItems(body []byte) []json.RawMessage {
	trimmed := []byte(strings.TrimSpace(string(body)))
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err == nil {
			return arr
		}
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

func cloneWithQuery(prevReq *http.Request, mutate func(q url.Values)) *http.Request {
	next := prevReq.Clone(prevReq.Context())
	q := next.URL.Query()
	mutate(q)
	next.URL.RawQuery = q.Encode()
	return next
}

// nonePaginator: single page, always done.
type nonePaginator struct{}

func (nonePaginator) Next(_ *http.Request, _ *http.Response, body []byte) (*http.Request, []json.RawMessage, bool, error) {
	return nil, ExtractItems(body), true, nil
}

// offsetLimitPaginator advances ?offset by ?limit; done when a page has
// fewer than limit items.
type offsetLimitPaginator struct{}

func (offsetLimitPaginator) Next(prevReq *http.Request, _ *http.Response, body []byte) (*http.Request, []json.RawMessage, bool, error) {
	q := prevReq.URL.Query()
	limit := atoiDefault(q.Get("limit"), 100)
	offset := atoiDefault(q.Get("offset"), 0)
	items := ExtractItems(body)
	if len(items) < limit {
		return nil, items, true, nil
	}
	next := cloneWithQuery(prevReq, func(q url.Values) {
		q["offset"] = []string{strconv.Itoa(offset + limit)}
		q["limit"] = []string{strconv.Itoa(limit)}
	})
	return next, items, false, nil
}

// pageNumberPaginator advances ?page; done when the page is empty or the
// body's total_pages says we are on the last page.
type pageNumberPaginator struct{}

func (pageNumberPaginator) Next(prevReq *http.Request, _ *http.Response, body []byte) (*http.Request, []json.RawMessage, bool, error) {
	q := prevReq.URL.Query()
	page := atoiDefault(q.Get("page"), 1)
	items := ExtractItems(body)
	if len(items) == 0 {
		return nil, items, true, nil
	}
	var meta struct {
		TotalPages int `json:"total_pages"`
	}
	_ = json.Unmarshal(body, &meta)
	if meta.TotalPages > 0 && page >= meta.TotalPages {
		return nil, items, true, nil
	}
	next := cloneWithQuery(prevReq, func(q url.Values) {
		q["page"] = []string{strconv.Itoa(page + 1)}
	})
	return next, items, false, nil
}

// cursorPaginator follows a next_cursor/next/cursor body field via the
// ?cursor query parameter; done when absent.
type cursorPaginator struct{}

func (cursorPaginator) Next(prevReq *http.Request, _ *http.Response, body []byte) (*http.Request, []json.RawMessage, bool, error) {
	items := ExtractItems(body)
	var meta map[string]json.RawMessage
	_ = json.Unmarshal(body, &meta)
	cursor := ""
	for _, key := range []string{"next_cursor", "next", "cursor"} {
		raw, ok := meta[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && s != "" {
			cursor = s
			break
		}
	}
	if cursor == "" {
		return nil, items, true, nil
	}
	next := cloneWithQuery(prevReq, func(q url.Values) {
		q["cursor"] = []string{cursor}
	})
	return next, items, false, nil
}

// linkHeaderPaginator follows RFC 5988 Link: <url>; rel="next" response
// headers; done when no next link is present.
type linkHeaderPaginator struct{}

var linkNextRe = regexp.MustCompile(`<([^>]+)>\s*;[^,]*\brel="?next"?`)

func (linkHeaderPaginator) Next(prevReq *http.Request, resp *http.Response, body []byte) (*http.Request, []json.RawMessage, bool, error) {
	items := ExtractItems(body)
	if resp == nil {
		return nil, items, true, nil
	}
	m := linkNextRe.FindStringSubmatch(resp.Header.Get("Link"))
	if m == nil {
		return nil, items, true, nil
	}
	ref, err := prevReq.URL.Parse(m[1])
	if err != nil {
		return nil, items, true, fmt.Errorf("parse Link header URL %q: %w", m[1], err)
	}
	next := prevReq.Clone(prevReq.Context())
	next.URL = ref
	return next, items, false, nil
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
