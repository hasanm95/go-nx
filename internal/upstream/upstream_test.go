package upstream

import (
	"testing"

	"github.com/hasanm95/go-nx/internal/config"
)


func TestSelector(t *testing.T){
	upstreams := []string{"A", "B", "C"}

	nextItem := Selector(upstreams)

	if got := nextItem(); got != "A" {
		t.Errorf("expected A, got %s", got)
	}

	if got := nextItem(); got != "B" {
		t.Errorf("expected B, got %s", got)
	}

	if got := nextItem(); got != "C" {
		t.Errorf("expected C, got %s", got)
	}

	// Verify wrap-around loop back to start

	if got := nextItem(); got != "A" {
		t.Errorf("expected wrap-around to A, got %s", got)
	}
}

func TestSelector_Empty(t *testing.T) {
	upstreams := []string{}

	nextItem := Selector(upstreams)

	if got := nextItem(); got != "" {
		t.Errorf("expected empty string for zero-slice, got %s", got)
	}
}

type LookupEntry struct {
	name      string
	selected  string
	upstreams []config.UpstreamConfig
	wantOk    bool
	wantURL   string
}

func TestLookup(t *testing.T) {
	upstreams := []config.UpstreamConfig{
		{ID: "node1", URL: "http://localhost:9000"},
		{ID: "node2", URL: "http://localhost:9001"},
		{ID: "node3", URL: "http://localhost:9002"},
		{ID: "node4", URL: "http://localhost:9004"},
	}

	entries := []LookupEntry{
		{
			name:      "valid lookup",
			selected:  "node3",
			upstreams: upstreams,
			wantOk:    true,
			wantURL:   "http://localhost:9002",
		},
		{
			name:      "id not present in a non-empty list",
			selected:  "node99",
			upstreams: upstreams,
			wantOk:    false,
		},
		{
			name:      "empty selected id",
			selected:  "",
			upstreams: upstreams,
			wantOk:    false,
		},
		{
			name:      "empty upstreams list",
			selected:  "node1",
			upstreams: []config.UpstreamConfig{},
			wantOk:    false,
		},
	}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			url, ok := Lookup(entry.selected, entry.upstreams)

			if ok != entry.wantOk {
				t.Fatalf("got ok=%v, want ok=%v", ok, entry.wantOk)
			}

			if entry.wantOk && url != entry.wantURL {
				t.Errorf("expected %s, got %s", entry.wantURL, url)
			}
		})
	}
}

func TestBuildSelectors(t *testing.T) {
	paths := []config.PathConfig{
		{Path: "/", Upstreams: []string{"node1"}},
		{Path: "/api/v1/*", Upstreams: []string{"node1", "node2"}},
	}

	selectors := BuildSelectors(paths)

	if len(selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(selectors))
	}

	rootSelect, ok := selectors["/"]
	if !ok {
		t.Fatalf("expected a selector for \"/\"")
	}
	if got := rootSelect(); got != "node1" {
		t.Errorf("expected node1, got %s", got)
	}

	apiSelect, ok := selectors["/api/v1/*"]
	if !ok {
		t.Fatalf("expected a selector for \"/api/v1/*\"")
	}
	if got := apiSelect(); got != "node1" {
		t.Errorf("expected node1, got %s", got)
	}
	if got := apiSelect(); got != "node2" {
		t.Errorf("expected node2, got %s", got)
	}
}

func TestBuildSelectors_IndependentCounters(t *testing.T) {
	paths := []config.PathConfig{
		{Path: "/a", Upstreams: []string{"node1", "node2"}},
		{Path: "/b", Upstreams: []string{"node1", "node2"}},
	}

	selectors := BuildSelectors(paths)

	selectors["/a"]()
	selectors["/a"]()
	thirdA := selectors["/a"]() // should wrap back to node1

	firstB := selectors["/b"]() // independent counter, should still be node1

	if thirdA != "node1" {
		t.Errorf("expected /a's 3rd call to wrap to node1, got %s", thirdA)
	}
	if firstB != "node1" {
		t.Errorf("expected /b's 1st call to be node1 (independent of /a), got %s", firstB)
	}
}

func TestBuildSelectors_EmptyPaths(t *testing.T) {
	selectors := BuildSelectors(nil)

	if len(selectors) != 0 {
		t.Errorf("expected empty map for nil paths, got %d entries", len(selectors))
	}
}
