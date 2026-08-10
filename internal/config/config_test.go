package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	err := os.WriteFile(path, []byte(serverYML), 0644)

	if err != nil {
		t.Fatalf("[TEST] failed to write temp config: %v", err)
	}

	cfg, err := ParseConfig(path)

	if err != nil {
		t.Fatalf("[TEST] failed to parse config file: %v", err)
	}

	if cfg.Server.Listen != 8080 {
		t.Errorf("got %d, want %d", cfg.Server.Listen, 8080)
	}

	if cfg.Server.Workers != 4 {
		t.Errorf("got %d, want %d", cfg.Server.Workers, 4)
	}

	if len(cfg.Server.Upstreams) != 2 {
		t.Fatalf("got %d, want %d", len(cfg.Server.Upstreams), 2)
	}

	if cfg.Server.Upstreams[0].ID != "node1" {
		t.Errorf("got %s, want %s", cfg.Server.Upstreams[0].ID, "node1")
	}

	if cfg.Server.Upstreams[0].URL != "http://localhost:8000" {
		t.Errorf("got %s, want %s", cfg.Server.Upstreams[0].URL, "http://localhost:8000")
	}

	if cfg.Server.Upstreams[1].ID != "node2" {
		t.Errorf("got %s, want %s", cfg.Server.Upstreams[1].ID, "node2")
	}

	if cfg.Server.Upstreams[1].URL != "http://localhost:8001" {
		t.Errorf("got %s, want %s", cfg.Server.Upstreams[1].URL, "http://localhost:8001")
	}

	if len(cfg.Server.Paths) != 2 {
		t.Fatalf("got %d, want %d", len(cfg.Server.Paths), 2)
	}

	if cfg.Server.Paths[0].Path != "/" {
		t.Errorf("got %s, want %s", cfg.Server.Paths[0].Path, "/")
	}

	if len(cfg.Server.Paths[0].Upstreams) != 2 {
		t.Fatalf("got %d, want %d", len(cfg.Server.Paths[0].Upstreams), 2)
	}

	if cfg.Server.Paths[0].Upstreams[0] != "node1" {
		t.Errorf("got %s, want %s", cfg.Server.Paths[0].Upstreams[0], "node1")
	}

	if cfg.Server.Paths[0].Upstreams[1] != "node2" {
		t.Errorf("got %s, want %s", cfg.Server.Paths[0].Upstreams[1], "node2")
	}

	if cfg.Server.Paths[1].Path != "/admin/*" {
		t.Errorf("got %s, want %s", cfg.Server.Paths[1].Path, "/admin/*")
	}

	if len(cfg.Server.Paths[1].Upstreams) != 1 {
		t.Fatalf("got %d, want %d", len(cfg.Server.Paths[1].Upstreams), 1)
	}

	if cfg.Server.Paths[1].Upstreams[0] != "node2" {
		t.Errorf("got %s, want %s", cfg.Server.Paths[1].Upstreams[0], "node2")
	}

	if len(cfg.Server.Headers) != 1 {
		t.Fatalf("got %d, want %d", len(cfg.Server.Headers), 1)
	}

	if cfg.Server.Headers[0].Key != "X-Forwarded-For"{
		t.Errorf("got %s, want %s", cfg.Server.Headers[0].Key, "X-Forwarded-For")
	}

	if cfg.Server.Headers[0].Value != "$ip"{
		t.Errorf("got %s, want %s", cfg.Server.Headers[0].Value, "$ip")
	}
}