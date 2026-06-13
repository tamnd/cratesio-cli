package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) keywordsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keywords",
		Short: "List popular keywords on crates.io",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching keywords...")
			kws, err := a.client.Keywords(cmd.Context())
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(kws, len(kws))
		},
	}
}
