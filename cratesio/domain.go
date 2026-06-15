package cratesio

import (
	"context"
	"regexp"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes crates.io as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/cratesio-cli/cratesio"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host dereferences
// cratesio:// URIs by routing to the operations Register installs. The same
// Domain builds the standalone cratesio binary (see cli.NewApp), so the binary
// and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the crates.io driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "cratesio",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "crates",
			Short:  "Browse the crates.io Rust package registry",
			Long: `crates reads the crates.io Rust package registry through the public v1 API.
No authentication or API key required. Every request carries a descriptive
User-Agent so the server knows who is asking.

crates is an independent tool and is not affiliated with the Rust Foundation
or crates.io.`,
			Site: Host,
			Repo: "https://github.com/tamnd/cratesio-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		Summary: "Search crates by keyword",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchCrates)

	kit.Handle(app, kit.OpMeta{
		Name:     "crate",
		Group:    "read",
		Single:   true,
		Summary:  "Get crate details by name",
		URIType:  "crate",
		Resolver: true,
		Args:     []kit.Arg{{Name: "name", Help: "crate name (e.g. tokio, serde)"}},
	}, getCrate)

	kit.Handle(app, kit.OpMeta{
		Name:    "versions",
		Group:   "read",
		Summary: "List versions of a crate",
		Args:    []kit.Arg{{Name: "name", Help: "crate name"}},
	}, listVersions)

	kit.Handle(app, kit.OpMeta{
		Name:    "owners",
		Group:   "read",
		Summary: "List owners of a crate",
		Args:    []kit.Arg{{Name: "name", Help: "crate name"}},
	}, listOwners)

	kit.Handle(app, kit.OpMeta{
		Name:    "deps",
		Group:   "read",
		Summary: "List dependencies of the latest version of a crate",
		Args:    []kit.Arg{{Name: "name", Help: "crate name"}},
	}, listDeps)

	kit.Handle(app, kit.OpMeta{
		Name:    "top",
		Group:   "read",
		Summary: "List top crates by all-time download count",
	}, listTop)

	kit.Handle(app, kit.OpMeta{
		Name:    "categories",
		Group:   "read",
		Summary: "List crates.io categories",
	}, listCategories)

	kit.Handle(app, kit.OpMeta{
		Name:    "keywords",
		Group:   "read",
		Summary: "List popular crates.io keywords",
	}, listKeywords)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// ─── inputs ───────────────────────────────────────────────────────────────────

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type crateInput struct {
	Name   string  `kit:"arg" help:"crate name (e.g. tokio, serde)"`
	Client *Client `kit:"inject"`
}

type versionsInput struct {
	Name   string  `kit:"arg" help:"crate name"`
	Limit  int     `kit:"flag,inherit" help:"max versions" default:"10"`
	Client *Client `kit:"inject"`
}

type ownersInput struct {
	Name   string  `kit:"arg" help:"crate name"`
	Client *Client `kit:"inject"`
}

type depsInput struct {
	Name   string  `kit:"arg" help:"crate name"`
	Client *Client `kit:"inject"`
}

type topInput struct {
	Page   int     `kit:"flag" help:"page number (1-based)" default:"1"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"25"`
	Client *Client `kit:"inject"`
}

type categoriesInput struct {
	Limit  int     `kit:"flag,inherit" help:"max categories" default:"20"`
	Client *Client `kit:"inject"`
}

type keywordsInput struct {
	Limit  int     `kit:"flag,inherit" help:"max keywords" default:"50"`
	Client *Client `kit:"inject"`
}

// ─── handlers ─────────────────────────────────────────────────────────────────

func searchCrates(ctx context.Context, in searchInput, emit func(Crate) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	crates, err := in.Client.SearchCrates(ctx, in.Query, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, cr := range crates {
		if err := emit(cr); err != nil {
			return err
		}
	}
	return nil
}

func getCrate(ctx context.Context, in crateInput, emit func(*Crate) error) error {
	cr, err := in.Client.GetCrate(ctx, in.Name)
	if err != nil {
		return mapErr(err)
	}
	return emit(cr)
}

func listVersions(ctx context.Context, in versionsInput, emit func(Version) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	versions, err := in.Client.ListVersions(ctx, in.Name, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, v := range versions {
		if err := emit(v); err != nil {
			return err
		}
	}
	return nil
}

func listOwners(ctx context.Context, in ownersInput, emit func(Owner) error) error {
	owners, err := in.Client.Owners(ctx, in.Name)
	if err != nil {
		return mapErr(err)
	}
	for _, o := range owners {
		if err := emit(o); err != nil {
			return err
		}
	}
	return nil
}

func listDeps(ctx context.Context, in depsInput, emit func(Dep) error) error {
	deps, err := in.Client.Deps(ctx, in.Name)
	if err != nil {
		return mapErr(err)
	}
	for _, d := range deps {
		if err := emit(d); err != nil {
			return err
		}
	}
	return nil
}

func listTop(ctx context.Context, in topInput, emit func(Crate) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 25
	}
	crates, err := in.Client.Top(ctx, in.Page, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, cr := range crates {
		if err := emit(cr); err != nil {
			return err
		}
	}
	return nil
}

func listCategories(ctx context.Context, in categoriesInput, emit func(Category) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	cats, err := in.Client.ListCategories(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, cat := range cats {
		if err := emit(cat); err != nil {
			return err
		}
	}
	return nil
}

func listKeywords(ctx context.Context, in keywordsInput, emit func(Keyword) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	kws, err := in.Client.ListKeywords(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, kw := range kws {
		if err := emit(kw); err != nil {
			return err
		}
	}
	return nil
}

// ─── Resolver ─────────────────────────────────────────────────────────────────

// crateNameRE matches a valid crate name: lowercase letters, digits, hyphens, underscores.
var crateNameRE = regexp.MustCompile(`^[a-z0-9_-]+$`)

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` touch no network. A crate name goes to ("crate", name);
// anything else (a search query with spaces, uppercase letters) goes to ("query", input).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty cratesio reference")
	}
	if crateNameRE.MatchString(input) {
		return "crate", input, nil
	}
	return "query", input, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "crate":
		return "https://" + Host + "/crates/" + id, nil
	case "query":
		return "https://" + Host + "/search?q=" + id, nil
	default:
		return "", errs.Usage("cratesio has no resource type %q", uriType)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// mapErr converts a library error into the kit error kind that carries the right
// exit code.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if err == ErrNotFound {
		return errs.NotFound("%s", err.Error())
	}
	return err
}
