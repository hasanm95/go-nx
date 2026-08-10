package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Entry struct {
	name string
	yamlContent string
	wantErr bool
	want *Config
}

func TestParseConfig(t *testing.T) {
	var serverYML = `
server:
  listen: 8080
  workers: 4
  upstreams:
    - id: node1
      url: http://localhost:8000

    - id: node2
      url: http://localhost:8001

  paths:
    - path: /
      upstreams: 
        - node1
        - node2

    - path: /admin/*
      upstreams: 
        - node2

  headers:
    - key: X-Forwarded-For
      value: "$ip"
`

	var malformedServerYML = `
server:
  listen: "8080"
  workers: "4"
`


	entries := []Entry{
		{
			name:        "valid full config",
			yamlContent: serverYML,
			wantErr:     false,
			want: &Config{
				Server: ServerConfig{
					Listen:  8080,
					Workers: 4,
					Upstreams: []UpstreamConfig{
						{ID: "node1", URL: "http://localhost:8000"},
						{ID: "node2", URL: "http://localhost:8001"},
					},
					Paths: []PathConfig{
						{Path: "/", Upstreams: []string{"node1", "node2"}},
						{Path: "/admin/*", Upstreams: []string{"node2"}},
					},
					Headers: []HeaderConfig{
						{Key: "X-Forwarded-For", Value: "$ip"},
					},
				},
			},
		},
		{
			name:        "malformed config",
			yamlContent: malformedServerYML,
			wantErr:     true,
			want: nil,
		},
	}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yml")
			err := os.WriteFile(path, []byte(entry.yamlContent), 0644)

			if err != nil {
				t.Fatalf("[TEST] failed to write temp config: %v", err)
			}

			cfg, err := ParseConfig(path)

			if entry.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Fatalf("[TEST] failed to parse config file: %v", err)
				}

				diff := cmp.Diff(entry.want, cfg)
				if diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}