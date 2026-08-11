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
			name: "valid paths",
			paths: []config.PathConfig{
				{
					Path: "/",
					Upstreams: []string{"node1"},
				},
			},
			requestedPath: "/",
			wantOk: true,
			wantMatch: &config.PathConfig{Path: "/", Upstreams: []string{"node1"}},
		},
		{
			name: "invalid paths",
			paths: []config.PathConfig{
				{
					Path: "/",
					Upstreams: []string{"node1"},
				},
			},
			requestedPath: "/nonexistent", 
			wantOk: false, 
			wantMatch: nil,
		},
		{
			name: "Prefix paths",
			paths: []config.PathConfig{
				{
					Path: "/admin/*",
					Upstreams: []string{"node1"},
				},
			},
			requestedPath: "/admin/settings", 
			wantOk: true, 
			wantMatch: &config.PathConfig{Path: "/admin/*", Upstreams: []string{"node1"}},
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