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

type LookupExpected struct {
	ok bool
	value string
}

type LookupEntry struct {
	name string
	selected string
	upstreams []config.UpstreamConfig
	expected LookupExpected
	wantErr bool
}

func TestLookup(t *testing.T) {
	upstreams := []config.UpstreamConfig{
		{
			ID: "node1",
			URL: "http://localhost:9000",
		},
		{
			ID: "node2",
			URL: "http://localhost:9001",
		},
		{
			ID: "node3",
			URL: "http://localhost:9002",
		},
		{
			ID: "node4",
			URL: "http://localhost:9004",
		},
	}
	entries := []LookupEntry{
		{
			name: "Valid lookup",
			selected: "node3",
			upstreams: upstreams,
			expected: LookupExpected{
				ok: true,
				value: "http://localhost:9002",
			},
			wantErr: false,
		},
		{
			name: "Unknown selected",
			selected: "",
			upstreams: upstreams,
			expected: LookupExpected{
				ok: false,
				value: "",
			},
			wantErr: true,
		},
	}

	for _, entry := range entries{
		t.Run(entry.name, func(t *testing.T) {
			url, _ := Lookup(entry.selected, upstreams)

			if entry.wantErr == false {
				if url != entry.expected.value {
					t.Errorf("exprected %s', got %s", entry.expected.value, url)
				}
			} else {
				if url != "" {
					t.Errorf("expected empty string for empty selected, got %s", url)
				}
			}
		})
	}

}
