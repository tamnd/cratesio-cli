// Package cli assembles the cratesio command tree from the cratesio domain
// on top of the any-cli/kit framework.
package cli

import (
	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/cratesio-cli/cratesio"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// NewApp assembles the kit application from the cratesio domain. The domain's
// Register installs the client factory and every operation, so the binary and a
// host (ant, which blank-imports the package) share one source of truth.
// kit.Run turns the App into the CLI, plus the serve and mcp surfaces and the
// typed-error-to-exit-code mapping.
func NewApp() *kit.App {
	id := cratesio.Domain{}.Info().Identity
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(func(cfg *kit.Config) {
		cfg.UserAgent = "tamnd-cratesio-cli/0.1 tamnd87@gmail.com"
	}))
	(cratesio.Domain{}).Register(app)
	app.AddCommand(newVersionCmd())
	return app
}
