package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) depsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps <name>",
		Short: "List crates that depend on a crate (reverse dependencies)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit := a.effectiveLimit(25)
			a.progressf("fetching reverse dependencies for %q...", args[0])
			rdeps, err := a.client.ReverseDeps(cmd.Context(), args[0], limit)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(rdeps, len(rdeps))
		},
	}
}
