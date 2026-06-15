package cratesio

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (mint, body, resolve), which need no network. The client's
// HTTP behaviour is covered in cratesio_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "cratesio" {
		t.Errorf("Scheme = %q, want cratesio", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "crates" {
		t.Errorf("Identity.Binary = %q, want crates", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"tokio", "crate", "tokio"},
		{"serde", "crate", "serde"},
		{"my-crate", "crate", "my-crate"},
		{"my_crate", "crate", "my_crate"},
		{"async runtime", "query", "async runtime"},
		{"web framework 2024", "query", "web framework 2024"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestLocateCrate(t *testing.T) {
	got, err := Domain{}.Locate("crate", "tokio")
	want := "https://" + Host + "/crates/tokio"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateQuery(t *testing.T) {
	got, err := Domain{}.Locate("query", "async runtime")
	want := "https://" + Host + "/search?q=async runtime"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip:
// a record mints to its URI, and a bare id resolves back to the same URI.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	cr := &Crate{ID: "tokio", Name: "tokio", Description: "An async runtime"}
	u, err := h.Mint(cr)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "cratesio://crate/tokio"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	got, err := h.ResolveOn("cratesio", "serde")
	if err != nil || got.String() != "cratesio://crate/serde" {
		t.Errorf("ResolveOn = (%q, %v), want cratesio://crate/serde", got.String(), err)
	}
}
