// Package cratesio is the library behind the cratesio command line:
// the HTTP client, request shaping, and the typed data models for crates.io.
//
// The crates.io v1 API is public and requires no authentication, but every
// request MUST carry a descriptive User-Agent or the server returns 403.
// Set it via Config.UserAgent or DefaultConfig(), which fills it in.
package cratesio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Host is the site this client talks to, and the host the URI driver claims.
const Host = "crates.io"

// baseURL is the root every request is built from.
const baseURL = "https://" + Host + "/api/v1"

// ErrNotFound is returned when the API returns a 404 or an empty result.
var ErrNotFound = errors.New("not found")

// Config holds all tunable parameters for a Client.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
	BaseURL   string // override for tests
}

// DefaultConfig returns a Config with sensible defaults.
// UserAgent is required by crates.io; the default identifies this client.
func DefaultConfig() Config {
	return Config{
		UserAgent: "tamnd-cratesio-cli/0.1 tamnd87@gmail.com",
		Rate:      300 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the crates.io v1 API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = baseURL
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// ─── output types ─────────────────────────────────────────────────────────────

// Crate is the primary record for a crates.io package.
type Crate struct {
	ID              string `kit:"id" json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Downloads       int64  `json:"downloads"`
	RecentDownloads int64  `json:"recent_downloads"`
	MaxVersion      string `json:"max_version"`
	NewestVersion   string `json:"newest_version"`
	Homepage        string `json:"homepage"`
	Repository      string `json:"repository"`
}

// Version is a record for a single published version of a crate.
type Version struct {
	ID        int    `kit:"id" json:"id"`
	CrateName string `json:"crate"`
	Num       string `json:"num"`
	Downloads int64  `json:"downloads"`
	Yanked    bool   `json:"yanked"`
	License   string `json:"license"`
	CreatedAt string `json:"created_at"`
}

// Category is a record for a crates.io category.
type Category struct {
	ID          string `kit:"id" json:"id"`
	Name        string `json:"category"`
	CratesCount int    `json:"crates_cnt"`
	Description string `json:"description"`
}

// ─── wire types ───────────────────────────────────────────────────────────────

type searchResponse struct {
	Crates []Crate `json:"crates"`
	Meta   struct {
		Total    int    `json:"total"`
		NextPage string `json:"next_page"`
	} `json:"meta"`
}

type crateResponse struct {
	Crate    Crate     `json:"crate"`
	Versions []Version `json:"versions"`
}

type versionsResponse struct {
	Versions []Version `json:"versions"`
}

type categoriesResponse struct {
	Categories []Category `json:"categories"`
}

// ─── client methods ───────────────────────────────────────────────────────────

// SearchCrates searches crates.io for crates matching query.
// limit controls the max results; capped at 100 per API page.
func (c *Client) SearchCrates(ctx context.Context, query string, limit int) ([]Crate, error) {
	if limit <= 0 {
		limit = 10
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("per_page", fmt.Sprintf("%d", pageSize))
	rawURL := c.cfg.BaseURL + "/crates?" + params.Encode()
	var resp searchResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if limit < len(resp.Crates) {
		resp.Crates = resp.Crates[:limit]
	}
	return resp.Crates, nil
}

// GetCrate fetches metadata for a single crate by name.
func (c *Client) GetCrate(ctx context.Context, name string) (*Crate, error) {
	rawURL := c.cfg.BaseURL + "/crates/" + url.PathEscape(name)
	var resp crateResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if resp.Crate.Name == "" {
		return nil, ErrNotFound
	}
	return &resp.Crate, nil
}

// ListVersions lists all published versions of a crate, truncated to limit.
func (c *Client) ListVersions(ctx context.Context, name string, limit int) ([]Version, error) {
	if limit <= 0 {
		limit = 10
	}
	rawURL := c.cfg.BaseURL + "/crates/" + url.PathEscape(name) + "/versions"
	var resp versionsResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if limit < len(resp.Versions) {
		resp.Versions = resp.Versions[:limit]
	}
	return resp.Versions, nil
}

// ListCategories lists crates.io categories, limited to limit results.
func (c *Client) ListCategories(ctx context.Context, limit int) ([]Category, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}
	params := url.Values{}
	params.Set("per_page", fmt.Sprintf("%d", pageSize))
	rawURL := c.cfg.BaseURL + "/categories?" + params.Encode()
	var resp categoriesResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if limit < len(resp.Categories) {
		resp.Categories = resp.Categories[:limit]
	}
	return resp.Categories, nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
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
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
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
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
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
