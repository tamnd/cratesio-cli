package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	var sort string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search crates.io",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(20)
			a.progressf("searching crates.io for %q...", args[0])
			crates, err := a.client.Search(cmd.Context(), args[0], sort, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(crates, len(crates))
		},
	}
	cmd.Flags().StringVar(&sort, "sort", "downloads", "sort order: downloads, alpha, new_crates, new_versions, recent_downloads")
	return cmd
}
