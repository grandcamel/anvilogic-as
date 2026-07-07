package cli

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/auth"
	"github.com/grandcamel/anvilogic-as/internal/config"
	"github.com/grandcamel/anvilogic-as/internal/httpx"
	"github.com/grandcamel/anvilogic-as/internal/mock"
	"github.com/grandcamel/anvilogic-as/internal/output"
)

const keyHint = "generate a key in the platform UI (Settings > Generate API Key), then run 'anvilogic-as auth set'"

func newAuthCmd(gf *globalFlags, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage the API credential",
	}
	cmd.AddCommand(newAuthSetCmd(gf), newAuthStatusCmd(gf), newAuthValidateCmd(gf, version))
	return cmd
}

func newAuthSetCmd(gf *globalFlags) *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store the API key (OS keychain, config-file fallback)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(apiKey)
			if key == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), "API key: ")
				line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				if err != nil && err != io.EOF {
					return output.Validation(fmt.Sprintf("read API key: %v", err), "")
				}
				key = strings.TrimSpace(line)
			}
			if key == "" {
				return output.Validation("empty API key", keyHint)
			}
			cfg := gf.resolveConfig()
			stored := "keyring"
			if err := (config.SystemKeyring{}).Set(config.KeyringService, cfg.Profile, key); err != nil {
				home, herr := os.UserHomeDir()
				if herr != nil {
					return output.Validation(herr.Error(), "")
				}
				if ferr := config.SetProfileKey(home, cfg.Profile, "api_key", key); ferr != nil {
					return output.Validation(fmt.Sprintf("store API key: keyring: %v; config file: %v", err, ferr), "")
				}
				stored = "config-file"
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "keyring unavailable (%v); stored in %s (0600)\n", err, config.Path(home))
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
				"stored":  stored,
				"profile": cfg.Profile,
				"api_key": config.Mask(key),
			})
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key value (prompted when omitted)")
	return cmd
}

func newAuthStatusCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which source provided the API key (masked)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := gf.resolveConfig()
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
				"profile":     cfg.Profile,
				"api_key":     config.Mask(cfg.APIKey),
				"source":      cfg.Sources["api_key"],
				"base_url":    cfg.BaseURL,
				"auth_scheme": cfg.AuthScheme,
			})
		},
	}
}

func newAuthValidateCmd(gf *globalFlags, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Verify that a usable API key is configured (probes the registry auth.probe endpoint when set)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := gf.resolveConfig()
			ts := auth.FromConfig(cfg)
			token, err := ts.Token(cmd.Context())
			if err != nil {
				return &output.Error{
					Code:    "auth",
					Message: "no API key configured",
					Hint:    keyHint,
					Exit:    output.ExitAuth,
				}
			}

			reg, rerr := gf.loadRegistry()
			if rerr != nil {
				return rerr
			}
			probeID := ""
			if reg.Auth != nil {
				probeID = reg.Auth.Probe
			}
			if probeID == "" {
				return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
					"status":  "key-present",
					"source":  cfg.Sources["api_key"],
					"api_key": config.Mask(cfg.APIKey),
					"note":    "registry has no auth.probe endpoint; remote validation skipped",
				})
			}
			probe, ok := reg.Endpoints[probeID]
			if !ok || !probe.HasKnownPath() {
				return output.Validation(
					fmt.Sprintf("registry auth.probe %q is missing or has no captured path", probeID),
					"fix the registry or wait for a later wave")
			}

			baseURL := cfg.BaseURL
			if baseURL == "" {
				baseURL = reg.BaseURL
			}
			method := probe.Method
			if strings.EqualFold(method, "UNKNOWN") || method == "" {
				method = http.MethodGet
			}
			header := http.Header{}
			header.Set("Authorization", auth.Header(cfg.AuthScheme, token))
			client := httpx.New(version, gf.timeout)
			if mock.Enabled() {
				client.HTTP.Transport = mock.NewTransport(reg, mock.FixturesDir())
			}
			resp, derr := client.Do(cmd.Context(), strings.ToUpper(method), strings.TrimRight(baseURL, "/")+probe.Path, header, nil)
			if derr != nil {
				return &output.Error{Code: "server", Message: derr.Error(), Exit: output.ExitServer}
			}
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				return output.FromResponse(resp.StatusCode, b)
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
				"status":  "valid",
				"probe":   probeID,
				"api_key": config.Mask(cfg.APIKey),
				"source":  cfg.Sources["api_key"],
			})
		},
	}
}
