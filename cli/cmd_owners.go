package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) ownersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "owners <name>",
		Short: "List owners of a crate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching owners for %q...", args[0])
			owners, err := a.client.Owners(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(owners, len(owners))
		},
	}
}
