package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/output"
)

func newRegistryCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Inspect and validate the endpoint registry",
	}
	cmd.AddCommand(newRegistryListCmd(gf), newRegistryDescribeCmd(gf), newRegistryValidateCmd(gf))
	return cmd
}

func newRegistryListCmd(gf *globalFlags) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registry endpoint ids",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := gf.loadRegistry()
			if err != nil {
				return err
			}
			ids := reg.IDs(domain)
			if gf.output == output.ModeText {
				for _, id := range ids {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), id)
				}
				return nil
			}
			type row struct {
				ID         string `json:"id"`
				Domain     string `json:"domain"`
				Method     string `json:"method"`
				Path       string `json:"path"`
				Risk       string `json:"risk"`
				Confidence string `json:"confidence"`
				Summary    string `json:"summary"`
			}
			rows := make([]row, 0, len(ids))
			for _, id := range ids {
				e := reg.Endpoints[id]
				rows = append(rows, row{id, e.Domain, e.Method, e.Path, e.Risk, e.Confidence, e.Summary})
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, rows)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "filter by domain")
	return cmd
}

func newRegistryDescribeCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "describe <id>",
		Short: "Show the full registry entry, including evidence and confidence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := gf.loadRegistry()
			if err != nil {
				return err
			}
			e, ok := reg.Endpoints[args[0]]
			if !ok {
				return &output.Error{
					Code:    "not-found",
					Message: fmt.Sprintf("no registry endpoint %q", args[0]),
					Hint:    "list ids with 'anvilogic-as registry list'",
					Exit:    output.ExitNotFound,
				}
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, e)
		},
	}
}

func newRegistryValidateCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate registry schema and enum values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := gf.loadRegistry()
			if err != nil {
				return err
			}
			errs := reg.Validate()
			if len(errs) > 0 {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Error()
				}
				return output.Validation(
					fmt.Sprintf("registry invalid: %d problem(s): %v", len(errs), msgs),
					"fix the registry YAML or pass a corrected --registry file")
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]any{
				"valid":     true,
				"version":   reg.Version,
				"base_url":  reg.BaseURL,
				"endpoints": len(reg.Endpoints),
			})
		},
	}
}
