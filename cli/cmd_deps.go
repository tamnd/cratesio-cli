package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) depsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps <name>",
		Short: "Show dependencies of the latest version of a crate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching dependencies for %q...", args[0])
			deps, err := a.client.Deps(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(deps, len(deps))
		},
	}
}
