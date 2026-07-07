package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/auth"
	"github.com/grandcamel/anvilogic-as/internal/httpx"
	"github.com/grandcamel/anvilogic-as/internal/mock"
	"github.com/grandcamel/anvilogic-as/internal/output"
	"github.com/grandcamel/anvilogic-as/internal/paginate"
	"github.com/grandcamel/anvilogic-as/internal/registry"
)

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

func newAPICmd(gf *globalFlags, version string) *cobra.Command {
	var (
		queries       []string
		data          string
		doPaginate    bool
		paginateStyle string
	)
	cmd := &cobra.Command{
		Use:   "api <METHOD> <path-or-registry-id>",
		Short: "Perform an API request against a raw path or a registry endpoint id",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(cmd, gf, version, args, queries, data, doPaginate, paginateStyle)
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&queries, "query", nil, "query parameter k=v (repeatable)")
	f.StringVar(&data, "data", "", "request body: inline JSON or @file")
	f.BoolVar(&doPaginate, "paginate", false, "follow pagination and emit all items")
	f.StringVar(&paginateStyle, "paginate-style", "", "pagination style: none|offset_limit|page_number|cursor|link_header")
	return cmd
}

func runAPI(cmd *cobra.Command, gf *globalFlags, version string, args, queries []string, data string, doPaginate bool, paginateStyle string) error {
	method := strings.ToUpper(args[0])
	if !httpMethods[method] {
		return output.Validation(
			fmt.Sprintf("invalid HTTP method %q", args[0]),
			"use one of GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
	}

	reg, err := gf.loadRegistry()
	if err != nil {
		return err
	}

	target := args[1]
	path := target
	var entry *registry.Endpoint
	if !strings.HasPrefix(target, "/") {
		e, ok := reg.Endpoints[target]
		if !ok {
			return output.Validation(
				fmt.Sprintf("unknown registry endpoint id %q", target),
				"pass an explicit path starting with '/', or list ids with 'anvilogic-as registry list'")
		}
		if !e.HasKnownPath() {
			return output.Validation(
				fmt.Sprintf("registry endpoint %q has no captured path", target),
				"endpoint path not yet captured (Wave 3); pass an explicit path")
		}
		entry = &e
		path = e.Path
	}

	cfg := gf.resolveConfig()
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = reg.BaseURL
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return output.Validation(fmt.Sprintf("invalid URL: %v", err), "check --base-url and the path")
	}
	q := u.Query()
	for _, kv := range queries {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return output.Validation(fmt.Sprintf("invalid --query %q", kv), "use --query key=value")
		}
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()

	var body []byte
	if data != "" {
		if rest, ok := strings.CutPrefix(data, "@"); ok {
			b, rerr := readDataFile(cmd, rest)
			if rerr != nil {
				return output.Validation(fmt.Sprintf("read --data file: %v", rerr), "")
			}
			body = b
		} else {
			body = []byte(data)
		}
	}

	header := http.Header{}
	if body != nil {
		header.Set("Content-Type", "application/json")
	}
	ts := auth.FromConfig(cfg)
	if token, terr := ts.Token(cmd.Context()); terr == nil {
		header.Set("Authorization", auth.Header(cfg.AuthScheme, token))
	}

	client := httpx.New(version, gf.timeout)
	if mock.Enabled() {
		client.HTTP.Transport = mock.NewTransport(reg, mock.FixturesDir())
	}

	if doPaginate {
		style := paginateStyle
		if style == "" && entry != nil {
			style = entry.Pagination.Style
		}
		pag, perr := paginate.ForStyle(style)
		if perr != nil {
			return output.Validation(
				perr.Error(),
				"--paginate requires --paginate-style (none|offset_limit|page_number|cursor|link_header) when the registry style is unknown")
		}
		items, ierr := paginateAll(cmd, client, pag, method, u.String(), header, body)
		if ierr != nil {
			return ierr
		}
		return output.PrintItems(cmd.OutOrStdout(), gf.output, items)
	}

	resp, err := client.Do(cmd.Context(), method, u.String(), header, body)
	if err != nil {
		return &output.Error{Code: "server", Message: err.Error(), Exit: output.ExitServer}
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return &output.Error{Code: "server", Message: err.Error(), Exit: output.ExitServer}
	}
	if resp.StatusCode >= 400 {
		return output.FromResponse(resp.StatusCode, b)
	}
	return output.PrintData(cmd.OutOrStdout(), gf.output, b)
}

func paginateAll(cmd *cobra.Command, client *httpx.Client, pag paginate.Paginator, method, firstURL string, header http.Header, body []byte) ([]json.RawMessage, error) {
	req, err := http.NewRequestWithContext(cmd.Context(), method, firstURL, nil)
	if err != nil {
		return nil, output.Validation(err.Error(), "")
	}
	var all []json.RawMessage
	for {
		resp, derr := client.Do(cmd.Context(), method, req.URL.String(), header, body)
		if derr != nil {
			return nil, &output.Error{Code: "server", Message: derr.Error(), Exit: output.ExitServer}
		}
		b, rerr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if rerr != nil {
			return nil, &output.Error{Code: "server", Message: rerr.Error(), Exit: output.ExitServer}
		}
		if resp.StatusCode >= 400 {
			return nil, output.FromResponse(resp.StatusCode, b)
		}
		next, items, done, perr := pag.Next(req, resp, b)
		if perr != nil {
			return nil, output.Validation(fmt.Sprintf("pagination: %v", perr), "")
		}
		all = append(all, items...)
		if done || next == nil {
			return all, nil
		}
		req = next
	}
}

func readDataFile(cmd *cobra.Command, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}
