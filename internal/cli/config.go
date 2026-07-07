package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/config"
	"github.com/grandcamel/anvilogic-as/internal/output"
)

func newConfigCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and write CLI configuration",
	}
	cmd.AddCommand(newConfigGetCmd(gf), newConfigSetCmd(gf), newConfigListCmd(gf))
	return cmd
}

func resolvedKey(cfg *config.Resolved, key string) (string, bool) {
	switch key {
	case "profile":
		return cfg.Profile, true
	case "base_url":
		return cfg.BaseURL, true
	case "auth_scheme":
		return cfg.AuthScheme, true
	case "api_key":
		return config.Mask(cfg.APIKey), true
	}
	return "", false
}

func newConfigGetCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print one resolved config value (profile|base_url|auth_scheme|api_key)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := gf.resolveConfig()
			v, ok := resolvedKey(cfg, args[0])
			if !ok {
				return output.Validation(fmt.Sprintf("unknown config key %q", args[0]),
					"valid keys: profile, base_url, auth_scheme, api_key")
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
				"key": args[0], "value": v, "source": cfg.Sources[args[0]],
			})
		},
	}
}

func newConfigSetCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Persist a config value to ~/.config/anvilogic-as/config.json",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			home, err := os.UserHomeDir()
			if err != nil {
				return output.Validation(err.Error(), "")
			}
			switch key {
			case "profile":
				err = config.SetDefaultProfile(home, value)
			case "base_url", "auth_scheme", "api_key":
				cfg := gf.resolveConfig()
				err = config.SetProfileKey(home, cfg.Profile, key, value)
			default:
				return output.Validation(fmt.Sprintf("unknown config key %q", key),
					"valid keys: profile, base_url, auth_scheme, api_key")
			}
			if err != nil {
				return output.Validation(err.Error(), "")
			}
			shown := value
			if key == "api_key" {
				shown = config.Mask(value)
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{
				"key": key, "value": shown, "file": config.Path(home),
			})
		},
	}
}

func newConfigListCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the resolved configuration with per-key sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := gf.resolveConfig()
			type kv struct {
				Key    string `json:"key"`
				Value  string `json:"value"`
				Source string `json:"source"`
			}
			var rows []kv
			for _, key := range []string{"profile", "base_url", "auth_scheme", "api_key"} {
				v, _ := resolvedKey(cfg, key)
				rows = append(rows, kv{key, v, cfg.Sources[key]})
			}
			if gf.output == output.ModeText {
				for _, r := range rows {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s (%s)\n", r.Key, r.Value, r.Source)
				}
				return nil
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, rows)
		},
	}
}
