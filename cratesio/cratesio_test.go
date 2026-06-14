package cratesio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.BaseURL = srv.URL
	return NewClient(cfg)
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/crates") {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent header")
		}
		_, _ = w.Write([]byte(`{
			"crates": [
				{"id":"serde","name":"serde","description":"A serialization framework","downloads":500000000,"recent_downloads":8000000,"max_version":"1.0.219","updated_at":"2024-12-01T10:00:00Z","homepage":"https://serde.rs","repository":""},
				{"id":"tokio","name":"tokio","description":"Async runtime","downloads":300000000,"recent_downloads":5000000,"max_version":"1.40.0","updated_at":"2024-11-01T10:00:00Z","homepage":"","repository":"https://github.com/tokio-rs/tokio"}
			]
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	crates, err := c.Search(context.Background(), "serde", "downloads", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(crates) != 2 {
		t.Fatalf("got %d crates, want 2", len(crates))
	}
	if crates[0].Name != "serde" {
		t.Errorf("crates[0].Name = %q, want serde", crates[0].Name)
	}
	if crates[0].Rank != 1 {
		t.Errorf("crates[0].Rank = %d, want 1", crates[0].Rank)
	}
	if crates[0].URL != "https://serde.rs" {
		t.Errorf("crates[0].URL = %q, want homepage", crates[0].URL)
	}
	if crates[1].Name != "tokio" {
		t.Errorf("crates[1].Name = %q, want tokio", crates[1].Name)
	}
	if crates[1].URL != "https://github.com/tokio-rs/tokio" {
		t.Errorf("crates[1].URL = %q, want repository fallback", crates[1].URL)
	}
}

func TestCrate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"crate": {
				"id":"serde","name":"serde","description":"A serialization framework",
				"downloads":500000000,"recent_downloads":8000000,
				"max_version":"1.0.219","updated_at":"2024-12-01T10:00:00Z",
				"homepage":"https://serde.rs","repository":"https://github.com/serde-rs/serde",
				"exact_match":true
			}
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	cr, err := c.Crate(context.Background(), "serde")
	if err != nil {
		t.Fatal(err)
	}
	if cr.Name != "serde" {
		t.Errorf("Name = %q, want serde", cr.Name)
	}
	if cr.MaxVersion != "1.0.219" {
		t.Errorf("MaxVersion = %q, want 1.0.219", cr.MaxVersion)
	}
	if cr.Downloads != 500000000 {
		t.Errorf("Downloads = %d, want 500000000", cr.Downloads)
	}
}

func TestCrateNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Crate(context.Background(), "nonexistent-crate-xyz")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"versions": [
				{"id":1,"num":"1.0.219","downloads":200000,"created_at":"2024-12-01T10:00:00Z","yanked":false,"license":"MIT OR Apache-2.0","rust_version":"1.31"},
				{"id":2,"num":"1.0.218","downloads":150000,"created_at":"2024-11-01T10:00:00Z","yanked":false,"license":"MIT OR Apache-2.0","rust_version":"1.31"},
				{"id":3,"num":"1.0.100","downloads":100000,"created_at":"2023-01-01T10:00:00Z","yanked":true,"license":"MIT OR Apache-2.0","rust_version":""}
			]
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	versions, err := c.Versions(context.Background(), "serde")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Fatalf("got %d versions, want 3", len(versions))
	}
	if versions[0].Num != "1.0.219" {
		t.Errorf("versions[0].Num = %q, want 1.0.219", versions[0].Num)
	}
	if !versions[2].IsYanked {
		t.Error("versions[2].IsYanked should be true")
	}
}

func TestOwners(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"users": [
				{"login":"dtolnay","name":"David Tolnay","kind":"user","url":"https://github.com/dtolnay"},
				{"login":"crate-io","name":"Crate Team","kind":"team","url":"https://github.com/crate-io"}
			]
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	owners, err := c.Owners(context.Background(), "serde")
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 2 {
		t.Fatalf("got %d owners, want 2", len(owners))
	}
	if owners[0].Login != "dtolnay" {
		t.Errorf("owners[0].Login = %q, want dtolnay", owners[0].Login)
	}
	if owners[1].Kind != "team" {
		t.Errorf("owners[1].Kind = %q, want team", owners[1].Kind)
	}
}

func TestDeps(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.HasSuffix(r.URL.Path, "/dependencies") {
			_, _ = w.Write([]byte(`{
				"dependencies": [
					{"crate_id":"proc-macro2","req":"^1.0","kind":"normal","optional":false,"features":["default"]},
					{"crate_id":"quote","req":"^1.0","kind":"normal","optional":false,"features":[]},
					{"crate_id":"syn","req":"^2.0","kind":"dev","optional":false,"features":[]}
				]
			}`))
			return
		}
		// crate lookup for max_version
		_, _ = w.Write([]byte(`{
			"crate": {
				"id":"serde","name":"serde","description":"serialization","downloads":1,"recent_downloads":1,
				"max_version":"1.0.219","updated_at":"2024-12-01T10:00:00Z",
				"homepage":"https://serde.rs","repository":""
			}
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	deps, err := c.Deps(context.Background(), "serde")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 3 {
		t.Fatalf("got %d deps, want 3", len(deps))
	}
	if deps[0].Name != "proc-macro2" {
		t.Errorf("deps[0].Name = %q, want proc-macro2", deps[0].Name)
	}
	if deps[0].Features != "default" {
		t.Errorf("deps[0].Features = %q, want default", deps[0].Features)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 HTTP calls (crate + deps), got %d", calls)
	}
}

func TestTop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("sort") != "downloads" {
			t.Errorf("sort = %q, want downloads", q.Get("sort"))
		}
		_, _ = w.Write([]byte(`{
			"crates": [
				{"id":"syn","name":"syn","description":"Parser","downloads":1000000000,"recent_downloads":15000000,"max_version":"2.0.91","updated_at":"2024-11-15T10:00:00Z","homepage":"","repository":"https://github.com/dtolnay/syn"},
				{"id":"serde","name":"serde","description":"Serialization","downloads":900000000,"recent_downloads":12000000,"max_version":"1.0.219","updated_at":"2024-12-01T10:00:00Z","homepage":"https://serde.rs","repository":""}
			]
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	crates, err := c.Top(context.Background(), 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(crates) != 2 {
		t.Fatalf("got %d crates, want 2", len(crates))
	}
	if crates[0].Rank != 1 {
		t.Errorf("crates[0].Rank = %d, want 1", crates[0].Rank)
	}
	if crates[1].Rank != 2 {
		t.Errorf("crates[1].Rank = %d, want 2", crates[1].Rank)
	}
}

func TestReverseDeps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/reverse_dependencies") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{
			"dependencies": [
				{"crate_id":"serde_derive","req":"^1.0","kind":"normal","downloads":999000000},
				{"crate_id":"serde_json","req":"^1.0","kind":"normal","downloads":876000000},
				{"crate_id":"bincode","req":"^1.3","kind":"normal","downloads":123000000}
			],
			"meta":{"total":34567}
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	rdeps, err := c.ReverseDeps(context.Background(), "serde", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rdeps) != 3 {
		t.Fatalf("got %d reverse deps, want 3", len(rdeps))
	}
	if rdeps[0].Name != "serde_derive" {
		t.Errorf("rdeps[0].Name = %q, want serde_derive", rdeps[0].Name)
	}
	if rdeps[0].Rank != 1 {
		t.Errorf("rdeps[0].Rank = %d, want 1", rdeps[0].Rank)
	}
	if rdeps[0].Downloads != 999000000 {
		t.Errorf("rdeps[0].Downloads = %d, want 999000000", rdeps[0].Downloads)
	}
	if rdeps[1].Name != "serde_json" {
		t.Errorf("rdeps[1].Name = %q, want serde_json", rdeps[1].Name)
	}
}

func TestReverseDepsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"dependencies": [
				{"crate_id":"a","req":"^1.0","kind":"normal","downloads":100},
				{"crate_id":"b","req":"^1.0","kind":"normal","downloads":90},
				{"crate_id":"c","req":"^1.0","kind":"normal","downloads":80}
			],
			"meta":{"total":3}
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	rdeps, err := c.ReverseDeps(context.Background(), "foo", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rdeps) != 2 {
		t.Errorf("got %d reverse deps, want 2 (limit applied)", len(rdeps))
	}
}

func TestCategoriesAndKeywords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/categories") {
			_, _ = w.Write([]byte(`{
				"categories": [
					{"slug":"web-programming","category":"Web programming","description":"Web crates","crates_cnt":3000},
					{"slug":"async","category":"Asynchronous","description":"Async crates","crates_cnt":2500}
				]
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"keywords": [
				{"keyword":"async","crates_cnt":2500},
				{"keyword":"serde","crates_cnt":2000}
			]
		}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)

	cats, err := c.Categories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 {
		t.Fatalf("got %d categories, want 2", len(cats))
	}
	if cats[0].Slug != "web-programming" {
		t.Errorf("cats[0].Slug = %q, want web-programming", cats[0].Slug)
	}
	if cats[0].CratesCount != 3000 {
		t.Errorf("cats[0].CratesCount = %d, want 3000", cats[0].CratesCount)
	}

	kws, err := c.Keywords(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(kws) != 2 {
		t.Fatalf("got %d keywords, want 2", len(kws))
	}
	if kws[0].Keyword != "async" {
		t.Errorf("kws[0].Keyword = %q, want async", kws[0].Keyword)
	}
}

// isNotFound is a test helper (mirrors the one in cli/errors.go).
func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
