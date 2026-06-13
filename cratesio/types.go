package cratesio

import "strings"

// Crate is the primary record for a crate from crates.io.
// Used by search, top, and crate commands.
type Crate struct {
	Rank        int    `json:"rank"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Downloads   int64  `json:"downloads"`
	RecentDown  int64  `json:"recent_downloads"`
	MaxVersion  string `json:"max_version"`
	Updated     string `json:"updated"`
	URL         string `json:"url"`
}

// Version is a record for a single published version of a crate.
type Version struct {
	Num         string `json:"num"`
	Downloads   int64  `json:"downloads"`
	CreatedAt   string `json:"created_at"`
	IsYanked    bool   `json:"is_yanked"`
	License     string `json:"license"`
	RustVersion string `json:"rust_version"`
}

// Owner is a record for a crate owner (user or team).
type Owner struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	URL   string `json:"url"`
}

// Dep is a record for a single dependency of a crate version.
type Dep struct {
	Name     string `json:"name"`
	Req      string `json:"req"`
	Kind     string `json:"kind"`
	Optional bool   `json:"optional"`
	Features string `json:"features"`
}

// Category is a record for a crates.io category.
type Category struct {
	Slug        string `json:"slug"`
	Name        string `json:"category"`
	Description string `json:"description"`
	CratesCount int    `json:"crates_cnt"`
}

// Keyword is a record for a popular keyword on crates.io.
type Keyword struct {
	Keyword     string `json:"keyword"`
	CratesCount int    `json:"crates_cnt"`
}

// ─── wire types (unexported) ─────────────────────────────────────────────────

type wireCrate struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Downloads       int64  `json:"downloads"`
	RecentDownloads int64  `json:"recent_downloads"`
	MaxVersion      string `json:"max_version"`
	UpdatedAt       string `json:"updated_at"`
	Homepage        string `json:"homepage"`
	Repository      string `json:"repository"`
	ExactMatch      bool   `json:"exact_match"`
}

type wireVersion struct {
	ID          int    `json:"id"`
	Num         string `json:"num"`
	Downloads   int64  `json:"downloads"`
	CreatedAt   string `json:"created_at"`
	Yanked      bool   `json:"yanked"`
	License     string `json:"license"`
	RustVersion string `json:"rust_version"`
}

type wireOwner struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	URL   string `json:"url"`
}

type wireDep struct {
	CrateID  string   `json:"crate_id"`
	Req      string   `json:"req"`
	Kind     string   `json:"kind"`
	Optional bool     `json:"optional"`
	Features []string `json:"features"`
}

type wireCategory struct {
	Slug        string `json:"slug"`
	Category    string `json:"category"`
	Description string `json:"description"`
	CratesCount int    `json:"crates_cnt"`
}

type wireKeyword struct {
	Keyword     string `json:"keyword"`
	CratesCount int    `json:"crates_cnt"`
}

// ─── response envelopes ──────────────────────────────────────────────────────

type searchResp struct {
	Crates []wireCrate `json:"crates"`
}

type crateResp struct {
	Crate wireCrate `json:"crate"`
}

type versionsResp struct {
	Versions []wireVersion `json:"versions"`
}

type ownersResp struct {
	Users []wireOwner `json:"users"`
}

type depsResp struct {
	Dependencies []wireDep `json:"dependencies"`
}

type categoriesResp struct {
	Categories []wireCategory `json:"categories"`
}

type keywordsResp struct {
	Keywords []wireKeyword `json:"keywords"`
}

// ─── converters ──────────────────────────────────────────────────────────────

func wireCrateToCrate(w wireCrate, rank int) Crate {
	u := w.Homepage
	if u == "" {
		u = w.Repository
	}
	if u == "" {
		u = "https://crates.io/crates/" + w.Name
	}
	return Crate{
		Rank:        rank,
		Name:        w.Name,
		Description: w.Description,
		Downloads:   w.Downloads,
		RecentDown:  w.RecentDownloads,
		MaxVersion:  w.MaxVersion,
		Updated:     w.UpdatedAt,
		URL:         u,
	}
}

func wireVersionToVersion(w wireVersion) Version {
	return Version{
		Num:         w.Num,
		Downloads:   w.Downloads,
		CreatedAt:   w.CreatedAt,
		IsYanked:    w.Yanked,
		License:     w.License,
		RustVersion: w.RustVersion,
	}
}

func wireOwnerToOwner(w wireOwner) Owner {
	return Owner{
		Login: w.Login,
		Name:  w.Name,
		Kind:  w.Kind,
		URL:   w.URL,
	}
}

func wireDepToDep(w wireDep) Dep {
	return Dep{
		Name:     w.CrateID,
		Req:      w.Req,
		Kind:     w.Kind,
		Optional: w.Optional,
		Features: strings.Join(w.Features, ";"),
	}
}

func wireCategoryToCategory(w wireCategory) Category {
	return Category{
		Slug:        w.Slug,
		Name:        w.Category,
		Description: w.Description,
		CratesCount: w.CratesCount,
	}
}

func wireKeywordToKeyword(w wireKeyword) Keyword {
	return Keyword{
		Keyword:     w.Keyword,
		CratesCount: w.CratesCount,
	}
}
