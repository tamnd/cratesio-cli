// Package cratesio is the library behind the crates command: the HTTP client,
// request shaping, and the typed data models for crates.io.
//
// The crates.io v1 API is public and requires no authentication, but it does
// require a descriptive User-Agent header or it returns 403.
package cratesio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultBase = "https://crates.io/api/v1"

// DefaultUserAgent is the User-Agent sent on every request.
// crates.io returns 403 if this header is absent or generic.
const DefaultUserAgent = "cratesio-cli/0.1.0 (github.com/tamnd/cratesio-cli)"

// ErrNotFound is returned when the API returns a 404 or an empty crate.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for a Client.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	BaseURL   string // override for tests
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      300 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the crates.io v1 API.
type Client struct {
	httpClient *http.Client
	userAgent  string
	rate       time.Duration
	retries    int
	base       string
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBase
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
		base:       base,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── API methods ─────────────────────────────────────────────────────────────

// Search searches crates.io for crates matching query.
// sort is one of: downloads, alpha, new_crates, new_versions, recent_downloads.
// limit controls the max results returned.
func (c *Client) Search(ctx context.Context, query, sort string, limit int) ([]Crate, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("sort", sort)
	params.Set("per_page", fmt.Sprintf("%d", pageSize))
	params.Set("page", "1")

	rawURL := c.base + "/crates?" + params.Encode()
	var resp searchResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	crates := make([]Crate, 0, len(resp.Crates))
	for i, wc := range resp.Crates {
		if i >= limit {
			break
		}
		crates = append(crates, wireCrateToCrate(wc, i+1))
	}
	return crates, nil
}

// Crate fetches metadata for a single crate by name.
func (c *Client) Crate(ctx context.Context, name string) (Crate, error) {
	rawURL := c.base + "/crates/" + url.PathEscape(name)
	var resp crateResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Crate{}, err
	}
	if resp.Crate.Name == "" {
		return Crate{}, ErrNotFound
	}
	return wireCrateToCrate(resp.Crate, 0), nil
}

// Versions lists all published versions of a crate (newest first).
func (c *Client) Versions(ctx context.Context, name string) ([]Version, error) {
	rawURL := c.base + "/crates/" + url.PathEscape(name) + "/versions"
	var resp versionsResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	versions := make([]Version, len(resp.Versions))
	for i, wv := range resp.Versions {
		versions[i] = wireVersionToVersion(wv)
	}
	return versions, nil
}

// Owners lists the owners of a crate (users and teams).
func (c *Client) Owners(ctx context.Context, name string) ([]Owner, error) {
	rawURL := c.base + "/crates/" + url.PathEscape(name) + "/owners"
	var resp ownersResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	owners := make([]Owner, len(resp.Users))
	for i, wo := range resp.Users {
		owners[i] = wireOwnerToOwner(wo)
	}
	return owners, nil
}

// Deps returns the dependencies of a crate's latest version.
// It first resolves the max_version via a Crate call, then fetches dependencies.
func (c *Client) Deps(ctx context.Context, name string) ([]Dep, error) {
	cr, err := c.Crate(ctx, name)
	if err != nil {
		return nil, err
	}
	version := cr.MaxVersion
	rawURL := c.base + "/crates/" + url.PathEscape(name) + "/" + url.PathEscape(version) + "/dependencies"
	var resp depsResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	deps := make([]Dep, len(resp.Dependencies))
	for i, wd := range resp.Dependencies {
		deps[i] = wireDepToDep(wd)
	}
	return deps, nil
}

// ReverseDeps returns crates that depend on the given crate (reverse dependencies).
// limit controls the max results; 0 uses the API default (100).
func (c *Client) ReverseDeps(ctx context.Context, name string, limit int) ([]ReverseDep, error) {
	params := url.Values{}
	pageSize := limit
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	params.Set("per_page", fmt.Sprintf("%d", pageSize))
	rawURL := c.base + "/crates/" + url.PathEscape(name) + "/reverse_dependencies?" + params.Encode()
	var resp reverseDepsResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	rdeps := make([]ReverseDep, 0, len(resp.Dependencies))
	for i, wd := range resp.Dependencies {
		if limit > 0 && i >= limit {
			break
		}
		rdeps = append(rdeps, wireReverseDep(wd, i+1))
	}
	return rdeps, nil
}

// Top returns the most downloaded crates on the given page (1-based).
// limit clips the returned slice.
func (c *Client) Top(ctx context.Context, page, limit int) ([]Crate, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("per_page", "100")
	params.Set("sort", "downloads")
	params.Set("page", fmt.Sprintf("%d", page))

	rawURL := c.base + "/crates?" + params.Encode()
	var resp searchResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	rankOffset := (page - 1) * 100
	crates := make([]Crate, 0, len(resp.Crates))
	for i, wc := range resp.Crates {
		if i >= limit {
			break
		}
		crates = append(crates, wireCrateToCrate(wc, rankOffset+i+1))
	}
	return crates, nil
}

// Categories lists all crates.io categories.
func (c *Client) Categories(ctx context.Context) ([]Category, error) {
	rawURL := c.base + "/categories?per_page=100"
	var resp categoriesResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	cats := make([]Category, len(resp.Categories))
	for i, wc := range resp.Categories {
		cats[i] = wireCategoryToCategory(wc)
	}
	return cats, nil
}

// Keywords lists popular keywords on crates.io.
func (c *Client) Keywords(ctx context.Context) ([]Keyword, error) {
	rawURL := c.base + "/keywords?per_page=100"
	var resp keywordsResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	kws := make([]Keyword, len(resp.Keywords))
	for i, wk := range resp.Keywords {
		kws[i] = wireKeywordToKeyword(wk)
	}
	return kws, nil
}
