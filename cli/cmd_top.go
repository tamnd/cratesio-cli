package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) topCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "top",
		Short: "List top downloaded crates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(25)
			a.progressf("fetching top crates (page %d)...", page)
			crates, err := a.client.Top(cmd.Context(), page, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(crates, len(crates))
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based, 100 crates per page)")
	return cmd
}
