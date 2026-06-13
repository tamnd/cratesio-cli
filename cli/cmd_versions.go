package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) versionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions <name>",
		Short: "List all versions of a crate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching versions for %q...", args[0])
			versions, err := a.client.Versions(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			n := a.effectiveLimit(len(versions))
			if n < len(versions) {
				versions = versions[:n]
			}
			return a.renderOrEmpty(versions, len(versions))
		},
	}
}
