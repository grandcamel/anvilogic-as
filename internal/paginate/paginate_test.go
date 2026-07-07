package paginate

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustReq(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func respWith(header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{StatusCode: 200, Header: header}
}

func ids(t *testing.T, items []json.RawMessage) []int {
	t.Helper()
	var out []int
	for _, it := range items {
		var v struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(it, &v); err != nil {
			t.Fatalf("bad item %s: %v", it, err)
		}
		out = append(out, v.ID)
	}
	return out
}

func checkIDs(t *testing.T, got []int, want ...int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}

func TestNonePaginator(t *testing.T) {
	pag, err := ForStyle("none")
	if err != nil {
		t.Fatal(err)
	}
	next, items, done, err := pag.Next(mustReq(t, "https://x/v1/items"), respWith(nil), fixture(t, "GET_v1_items.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !done || next != nil {
		t.Error("none paginator must finish after one page")
	}
	checkIDs(t, ids(t, items), 1, 2)
}

func TestOffsetLimitPaginatorWalksTwoPages(t *testing.T) {
	pag, err := ForStyle("offset_limit")
	if err != nil {
		t.Fatal(err)
	}
	req := mustReq(t, "https://x/v1/items?limit=2&offset=0")

	next, items, done, err := pag.Next(req, respWith(nil), fixture(t, "GET_v1_items.json"))
	if err != nil || done {
		t.Fatalf("page1: done=%v err=%v", done, err)
	}
	checkIDs(t, ids(t, items), 1, 2)
	if got := next.URL.Query().Get("offset"); got != "2" {
		t.Fatalf("next offset = %q, want 2", got)
	}

	next2, items2, done2, err := pag.Next(next, respWith(nil), fixture(t, "GET_v1_items_page2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !done2 || next2 != nil {
		t.Error("page2 (1 item < limit 2) must be final")
	}
	checkIDs(t, ids(t, items2), 3)
}

func TestPageNumberPaginatorWalksTwoPages(t *testing.T) {
	pag, err := ForStyle("page_number")
	if err != nil {
		t.Fatal(err)
	}
	req := mustReq(t, "https://x/v1/items?page=1")

	next, items, done, err := pag.Next(req, respWith(nil), fixture(t, "GET_v1_items.json"))
	if err != nil || done {
		t.Fatalf("page1: done=%v err=%v", done, err)
	}
	checkIDs(t, ids(t, items), 1, 2)
	if got := next.URL.Query().Get("page"); got != "2" {
		t.Fatalf("next page = %q, want 2", got)
	}

	_, items2, done2, err := pag.Next(next, respWith(nil), fixture(t, "GET_v1_items_page2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !done2 {
		t.Error("page 2 of total_pages 2 must be final")
	}
	checkIDs(t, ids(t, items2), 3)
}

func TestCursorPaginatorWalksTwoPages(t *testing.T) {
	pag, err := ForStyle("cursor")
	if err != nil {
		t.Fatal(err)
	}
	req := mustReq(t, "https://x/v1/items")

	next, items, done, err := pag.Next(req, respWith(nil), fixture(t, "GET_v1_items.json"))
	if err != nil || done {
		t.Fatalf("page1: done=%v err=%v", done, err)
	}
	checkIDs(t, ids(t, items), 1, 2)
	if got := next.URL.Query().Get("cursor"); got != "cur-2" {
		t.Fatalf("next cursor = %q, want cur-2", got)
	}

	next2, items2, done2, err := pag.Next(next, respWith(nil), fixture(t, "GET_v1_items_page2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !done2 || next2 != nil {
		t.Error("null next_cursor must finish pagination")
	}
	checkIDs(t, ids(t, items2), 3)
}

func TestLinkHeaderPaginatorWalksTwoPages(t *testing.T) {
	pag, err := ForStyle("link_header")
	if err != nil {
		t.Fatal(err)
	}
	req := mustReq(t, "https://x/v1/items")

	h := http.Header{}
	h.Set("Link", `</v1/items?page=2>; rel="next", </v1/items?page=9>; rel="last"`)
	next, items, done, err := pag.Next(req, respWith(h), fixture(t, "GET_v1_items.json"))
	if err != nil || done {
		t.Fatalf("page1: done=%v err=%v", done, err)
	}
	checkIDs(t, ids(t, items), 1, 2)
	if got := next.URL.String(); got != "https://x/v1/items?page=2" {
		t.Fatalf("next URL = %q", got)
	}

	next2, items2, done2, err := pag.Next(next, respWith(http.Header{}), fixture(t, "GET_v1_items_page2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !done2 || next2 != nil {
		t.Error("missing Link rel=next must finish pagination")
	}
	checkIDs(t, ids(t, items2), 3)
}

func TestForStyleRejectsUnknown(t *testing.T) {
	for _, style := range []string{"", "unknown", "bogus"} {
		if _, err := ForStyle(style); err == nil {
			t.Errorf("ForStyle(%q) should fail", style)
		}
	}
}

func TestCanonicalizeAliases(t *testing.T) {
	cases := map[string]string{
		"offsetLimit": StyleOffsetLimit,
		"offset":      StyleOffsetLimit,
		"page":        StylePageNumber,
		"link":        StyleLinkHeader,
		"CURSOR":      StyleCursor,
	}
	for in, want := range cases {
		if got := Canonicalize(in); got != want {
			t.Errorf("Canonicalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractItems(t *testing.T) {
	if got := ExtractItems([]byte(`[1,2,3]`)); len(got) != 3 {
		t.Errorf("array: %d items, want 3", len(got))
	}
	if got := ExtractItems([]byte(`{"data":[1,2]}`)); len(got) != 2 {
		t.Errorf("data envelope: %d items, want 2", len(got))
	}
	if got := ExtractItems([]byte(`{"status":"ok"}`)); len(got) != 1 {
		t.Errorf("plain object: %d items, want 1", len(got))
	}
}
