package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hasanm95/go-nx/internal/config"
	"github.com/hasanm95/go-nx/internal/proxy"
	"github.com/hasanm95/go-nx/internal/upstream"
)

func TestNewHandler_ProxiesSuccessfully(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("expected path /, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from upstream"))
	}))
	defer mockUpstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Upstreams: []config.UpstreamConfig{
				{ID: "node1", URL: mockUpstream.URL},
			},
			Paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
			},
		},
	}

	h := NewHandler(cfg, upstream.BuildSelectors(cfg.Server.Paths), proxy.NewProxy())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	h(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", res.Code)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "hello from upstream" {
		t.Errorf("expected body %q, got %q", "hello from upstream", string(body))
	}
}

func TestNewHandler_NoRuleMatched(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
			},
		},
	}

	h := NewHandler(cfg, upstream.BuildSelectors(cfg.Server.Paths), proxy.NewProxy())

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	res := httptest.NewRecorder()
	h(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.Code)
	}
}

func TestNewHandler_UpstreamUnreachable(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Upstreams: []config.UpstreamConfig{
				{ID: "node1", URL: "http://this-domain-does-not-exist-at-all-12345.xyz"},
			},
			Paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
			},
		},
	}

	h := NewHandler(cfg, upstream.BuildSelectors(cfg.Server.Paths), proxy.NewProxy())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	h(res, req)

	if res.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", res.Code)
	}
}