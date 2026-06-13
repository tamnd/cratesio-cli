package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/cratesio-cli/cratesio"
)

func (a *App) crateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "crate <name>",
		Short: "Show metadata for a crate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching crate %q...", args[0])
			cr, err := a.client.Crate(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]cratesio.Crate{cr})
		},
	}
}
