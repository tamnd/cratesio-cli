package cratesio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/cratesio-cli/cratesio"
)

func newTestClient(ts *httptest.Server) *cratesio.Client {
	cfg := cratesio.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return cratesio.NewClient(cfg)
}

func TestSearchCratesSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"crates":[{"id":"serde","name":"serde","description":"Serialization framework","downloads":500000000,"max_version":"1.0.219","newest_version":"1.0.219"}],"meta":{"total":1}}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.SearchCrates(context.Background(), "serde", 5)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("request carried no User-Agent header")
	}
	if !strings.Contains(gotUA, "tamnd-cratesio-cli") {
		t.Errorf("User-Agent = %q, want to contain tamnd-cratesio-cli", gotUA)
	}
}

func TestSearchCratesReturnsResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/crates") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"crates":[
			{"id":"serde","name":"serde","description":"A serialization framework","downloads":500000000,"recent_downloads":8000000,"max_version":"1.0.219","newest_version":"1.0.219","homepage":"https://serde.rs","repository":"https://github.com/serde-rs/serde"},
			{"id":"tokio","name":"tokio","description":"Async runtime","downloads":300000000,"recent_downloads":5000000,"max_version":"1.40.0","newest_version":"1.40.0","homepage":"https://tokio.rs","repository":"https://github.com/tokio-rs/tokio"}
		],"meta":{"total":2,"next_page":null}}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	crates, err := c.SearchCrates(context.Background(), "serde", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(crates) != 2 {
		t.Fatalf("len(crates) = %d, want 2", len(crates))
	}
	if crates[0].Name != "serde" {
		t.Errorf("crates[0].Name = %q, want serde", crates[0].Name)
	}
	if crates[0].Homepage != "https://serde.rs" {
		t.Errorf("crates[0].Homepage = %q, want https://serde.rs", crates[0].Homepage)
	}
	if crates[1].Name != "tokio" {
		t.Errorf("crates[1].Name = %q, want tokio", crates[1].Name)
	}
}

func TestSearchCratesLimitApplied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"crates":[
			{"id":"a","name":"a","downloads":1,"max_version":"1.0.0"},
			{"id":"b","name":"b","downloads":2,"max_version":"1.0.0"},
			{"id":"c","name":"c","downloads":3,"max_version":"1.0.0"}
		],"meta":{"total":3}}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	crates, err := c.SearchCrates(context.Background(), "a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(crates) != 2 {
		t.Errorf("len(crates) = %d, want 2 (limit applied)", len(crates))
	}
}

func TestGetCrateReturnsRecord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"crate":{
			"id":"tokio","name":"tokio","description":"An event-driven, non-blocking I/O platform",
			"downloads":735751219,"recent_downloads":12406893,
			"max_version":"1.52.3","newest_version":"1.52.3",
			"homepage":"https://tokio.rs","repository":"https://github.com/tokio-rs/tokio"
		},"versions":[]}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	cr, err := c.GetCrate(context.Background(), "tokio")
	if err != nil {
		t.Fatal(err)
	}
	if cr.Name != "tokio" {
		t.Errorf("Name = %q, want tokio", cr.Name)
	}
	if cr.MaxVersion != "1.52.3" {
		t.Errorf("MaxVersion = %q, want 1.52.3", cr.MaxVersion)
	}
	if cr.Downloads != 735751219 {
		t.Errorf("Downloads = %d, want 735751219", cr.Downloads)
	}
	if cr.Homepage != "https://tokio.rs" {
		t.Errorf("Homepage = %q, want https://tokio.rs", cr.Homepage)
	}
}

func TestGetCrateNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetCrate(context.Background(), "nonexistent-xyz-999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestListVersionsTruncated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"versions":[
			{"id":1,"crate":"serde","num":"1.0.219","downloads":200000,"created_at":"2024-12-01T10:00:00Z","yanked":false,"license":"MIT OR Apache-2.0"},
			{"id":2,"crate":"serde","num":"1.0.218","downloads":150000,"created_at":"2024-11-01T10:00:00Z","yanked":false,"license":"MIT OR Apache-2.0"},
			{"id":3,"crate":"serde","num":"1.0.100","downloads":100000,"created_at":"2023-01-01T10:00:00Z","yanked":true,"license":"MIT OR Apache-2.0"}
		]}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	versions, err := c.ListVersions(context.Background(), "serde", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("len(versions) = %d, want 2 (truncated)", len(versions))
	}
	if versions[0].Num != "1.0.219" {
		t.Errorf("versions[0].Num = %q, want 1.0.219", versions[0].Num)
	}
	if versions[1].Num != "1.0.218" {
		t.Errorf("versions[1].Num = %q, want 1.0.218", versions[1].Num)
	}
}

func TestListVersionsYanked(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"versions":[
			{"id":1,"crate":"serde","num":"1.0.219","downloads":200000,"created_at":"2024-12-01T10:00:00Z","yanked":false,"license":"MIT OR Apache-2.0"},
			{"id":3,"crate":"serde","num":"1.0.100","downloads":100000,"created_at":"2023-01-01T10:00:00Z","yanked":true,"license":"MIT OR Apache-2.0"}
		]}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	versions, err := c.ListVersions(context.Background(), "serde", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("len(versions) = %d, want 2", len(versions))
	}
	if versions[1].Yanked != true {
		t.Error("versions[1].Yanked should be true")
	}
}

func TestListCategories(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"categories":[
			{"id":"asynchronous","category":"Asynchronous","description":"Async runtimes and utilities","crates_cnt":2156},
			{"id":"web-programming","category":"Web programming","description":"Web frameworks","crates_cnt":3000}
		]}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	cats, err := c.ListCategories(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 {
		t.Fatalf("len(cats) = %d, want 2", len(cats))
	}
	if cats[0].ID != "asynchronous" {
		t.Errorf("cats[0].ID = %q, want asynchronous", cats[0].ID)
	}
	if cats[0].CratesCount != 2156 {
		t.Errorf("cats[0].CratesCount = %d, want 2156", cats[0].CratesCount)
	}
	if cats[1].Name != "Web programming" {
		t.Errorf("cats[1].Name = %q, want Web programming", cats[1].Name)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"crate":{"id":"serde","name":"serde","description":"x","downloads":1,"max_version":"1.0.0","newest_version":"1.0.0"},"versions":[]}`))
	}))
	defer ts.Close()

	cfg := cratesio.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := cratesio.NewClient(cfg)

	_, err := c.GetCrate(context.Background(), "serde")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
