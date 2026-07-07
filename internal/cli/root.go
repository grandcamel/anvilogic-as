// Package cli wires the cobra command tree. Commands return *output.Error
// for typed failures; Execute maps them to structured stderr JSON and the
// documented exit codes.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/config"
	"github.com/grandcamel/anvilogic-as/internal/httpx"
	"github.com/grandcamel/anvilogic-as/internal/output"
	"github.com/grandcamel/anvilogic-as/internal/registry"
)

type globalFlags struct {
	output       string
	profile      string
	timeout      time.Duration
	registryPath string
	baseURL      string
}

// Execute runs the CLI and returns the process exit code.
func Execute(ctx context.Context, version string) int {
	root := NewRootCmd(version)
	if err := root.ExecuteContext(ctx); err != nil {
		var oe *output.Error
		if errors.As(err, &oe) {
			output.WriteError(os.Stderr, oe)
			return oe.Exit
		}
		output.WriteError(os.Stderr, err)
		return output.ExitValidation
	}
	return output.ExitOK
}

// NewRootCmd builds the full command tree.
func NewRootCmd(version string) *cobra.Command {
	gf := &globalFlags{}
	root := &cobra.Command{
		Use:           "anvilogic-as",
		Short:         "Agent-first CLI for the Anvilogic SaaS control-plane API",
		Long:          "anvilogic-as is a minimalist, registry-driven CLI for the Anvilogic SaaS\ncontrol-plane API. The endpoint registry (api-surface/endpoints.yaml) is\nembedded in the binary; stdout carries data only, diagnostics go to stderr.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !output.ValidMode(gf.output) {
				return output.Validation(
					fmt.Sprintf("invalid --output %q", gf.output),
					"valid values: json, ndjson, text")
			}
			return nil
		},
	}
	pf := root.PersistentFlags()
	pf.StringVar(&gf.output, "output", output.ModeJSON, "output format: json|ndjson|text")
	pf.StringVar(&gf.profile, "profile", "", "config profile name")
	pf.DurationVar(&gf.timeout, "timeout", httpx.DefaultTimeout, "HTTP request timeout")
	pf.StringVar(&gf.registryPath, "registry", "", "path to a registry YAML overriding the embedded copy")
	pf.StringVar(&gf.baseURL, "base-url", "", "API base URL override")

	root.AddCommand(
		newAPICmd(gf, version),
		newAuthCmd(gf, version),
		newConfigCmd(gf),
		newRegistryCmd(gf),
		newVersionCmd(gf, version),
	)
	return root
}

func (gf *globalFlags) loadRegistry() (*registry.Registry, error) {
	reg, err := registry.Load(gf.registryPath)
	if err != nil {
		return nil, output.Validation(err.Error(), "check --registry / ANVILOGIC_REGISTRY_PATH")
	}
	return reg, nil
}

func (gf *globalFlags) resolveConfig() *config.Resolved {
	return config.Resolve(config.Options{
		ProfileFlag: gf.profile,
		BaseURLFlag: gf.baseURL,
	})
}
