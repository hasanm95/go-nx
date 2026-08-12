package matcher

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hasanm95/go-nx/internal/config"
)

type MatchEntry struct {
	name string
	paths []config.PathConfig
	requestedPath string
	wantMatch *config.PathConfig
	wantOk bool
}

func TestMatchPath(t *testing.T) {
	entries := []MatchEntry{
		{
			name: "exact match at root",
			paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
			},
			requestedPath: "/",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/", Upstreams: []string{"node1"}},
		},
		{
			name: "no rules match",
			paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
			},
			requestedPath: "/other",
			wantOk:        false,
			wantMatch:     nil,
		},
		{
			name:          "empty rule list",
			paths:         nil,
			requestedPath: "/",
			wantOk:        false,
			wantMatch:     nil,
		},
		{
			name: "prefix match",
			paths: []config.PathConfig{
				{Path: "/admin/*", Upstreams: []string{"node1"}},
			},
			requestedPath: "/admin/settings",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/admin/*", Upstreams: []string{"node1"}},
		},
		{
			name: "prefix does not match unrelated path",
			paths: []config.PathConfig{
				{Path: "/admin/*", Upstreams: []string{"node1"}},
			},
			requestedPath: "/other",
			wantOk:        false,
			wantMatch:     nil,
		},
		{
			name: "prefix does not false-positive on similar path",
			paths: []config.PathConfig{
				{Path: "/admin/*", Upstreams: []string{"node1"}},
			},
			// "/administrator" starts with "/admin" but NOT "/admin/" — must not match
			requestedPath: "/administrator",
			wantOk:        false,
			wantMatch:     nil,
		},
		{
			name: "exact beats prefix, prefix listed first",
			paths: []config.PathConfig{
				{Path: "/admin/*", Upstreams: []string{"node-wild"}},
				{Path: "/admin/settings", Upstreams: []string{"node-exact"}},
			},
			requestedPath: "/admin/settings",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/admin/settings", Upstreams: []string{"node-exact"}},
		},
		{
			name: "exact beats prefix, exact listed first",
			paths: []config.PathConfig{
				{Path: "/admin/settings", Upstreams: []string{"node-exact"}},
				{Path: "/admin/*", Upstreams: []string{"node-wild"}},
			},
			requestedPath: "/admin/settings",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/admin/settings", Upstreams: []string{"node-exact"}},
		},
		{
			name: "prefix rule does not match the bare prefix without trailing slash",
			paths: []config.PathConfig{
				{Path: "/admin/*", Upstreams: []string{"node1"}},
			},
			requestedPath: "/admin",
			wantOk:        false,
			wantMatch:     nil,
		},
		{
			name: "longest prefix wins, shorter listed first",
			paths: []config.PathConfig{
				{Path: "/api/*", Upstreams: []string{"node-general"}},
				{Path: "/api/v1/*", Upstreams: []string{"node-v1"}},
			},
			requestedPath: "/api/v1/users",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/api/v1/*", Upstreams: []string{"node-v1"}},
		},
		{
			name: "longest prefix wins, longer listed first",
			paths: []config.PathConfig{
				{Path: "/api/v1/*", Upstreams: []string{"node-v1"}},
				{Path: "/api/*", Upstreams: []string{"node-general"}},
			},
			requestedPath: "/api/v1/users",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/api/v1/*", Upstreams: []string{"node-v1"}},
		},
		{
			name: "root wildcard catches path with no more specific rule",
			paths: []config.PathConfig{
				{Path: "/*", Upstreams: []string{"node-catchall"}},
				{Path: "/admin/*", Upstreams: []string{"node-admin"}},
			},
			requestedPath: "/other",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/*", Upstreams: []string{"node-catchall"}},
		},
		{
			name: "more specific wildcard wins over root wildcard",
			paths: []config.PathConfig{
				{Path: "/*", Upstreams: []string{"node-catchall"}},
				{Path: "/admin/*", Upstreams: []string{"node-admin"}},
			},
			requestedPath: "/admin/settings",
			wantOk:        true,
			wantMatch:     &config.PathConfig{Path: "/admin/*", Upstreams: []string{"node-admin"}},
		},
	}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			match, ok := MatchPath(entry.paths, entry.requestedPath)

			if ok != entry.wantOk {
				t.Fatalf("got ok=%v, want ok=%v", ok, entry.wantOk)
			}

			if entry.wantOk {
				if diff := cmp.Diff(entry.wantMatch, match); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// Separate test: MatchPath should not mutate the caller's original slice,
// since newRouteConfig is supposed to sort a clone, not the input itself.
func TestMatchPath_DoesNotMutateInput(t *testing.T) {
	original := []config.PathConfig{
		{Path: "/admin/*", Upstreams: []string{"node-wild"}},
		{Path: "/admin/settings", Upstreams: []string{"node-exact"}},
	}
	// keep a separate copy of what "before" looked like
	before := append([]config.PathConfig(nil), original...)

	_, _ = MatchPath(original, "/admin/settings")

	if diff := cmp.Diff(before, original); diff != "" {
		t.Errorf("MatchPath mutated its input slice (-before +after):\n%s", diff)
	}
}