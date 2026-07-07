package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grandcamel/anvilogic-as/internal/output"
)

func newVersionCmd(gf *globalFlags, version string) *cobra.Command {
	var checkMin string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkMin != "" {
				ok, err := versionAtLeast(version, checkMin)
				if err != nil {
					return output.Validation(err.Error(), "use --check-min X.Y.Z")
				}
				if !ok {
					return output.Validation(
						fmt.Sprintf("version %s is below required minimum %s", version, checkMin),
						"upgrade anvilogic-as")
				}
			}
			if gf.output == output.ModeText {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
				return nil
			}
			return output.PrintValue(cmd.OutOrStdout(), gf.output, map[string]string{"version": version})
		},
	}
	cmd.Flags().StringVar(&checkMin, "check-min", "", "exit 1 if the CLI version is below X.Y.Z")
	return cmd
}

// versionAtLeast compares dotted versions. Dev/pseudo versions ("dev")
// cannot be compared and are treated as satisfying any minimum.
func versionAtLeast(have, min string) (bool, error) {
	h, herr := parseSemver(have)
	if herr != nil {
		return true, nil // dev build; cannot compare
	}
	m, merr := parseSemver(min)
	if merr != nil {
		return false, fmt.Errorf("invalid --check-min %q", min)
	}
	for i := 0; i < 3; i++ {
		if h[i] != m[i] {
			return h[i] > m[i], nil
		}
	}
	return true, nil
}

func parseSemver(v string) ([3]int, error) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return out, fmt.Errorf("invalid version %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("invalid version %q", v)
		}
		out[i] = n
	}
	return out, nil
}
