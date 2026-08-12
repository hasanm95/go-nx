package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hasanm95/go-nx/internal/config"
)

func validConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Upstreams: []config.UpstreamConfig{
				{ID: "node1", URL: "http://localhost:8000"},
				{ID: "node2", URL: "http://localhost:8000/api/v1/user"},
			},
			Paths: []config.PathConfig{
				{Path: "/", Upstreams: []string{"node1"}},
				{Path: "/api/v1/*", Upstreams: []string{"node1", "node2"}},
			},
		},
	}
}

type HandlerEntry struct {
	name string
	cfg *config.Config
	requestedPath string
	expectedCode int
	expectedBody string
}

func TestNewHandler(t *testing.T) {

	entries := []HandlerEntry{
		{
			name: "Sucess response",
			cfg: validConfig(),
			requestedPath: "/",
			expectedCode: 200,
			expectedBody: "matched rule: /, upstreams: [node1]",
		},
		{
			name: "Failed response",
			cfg: validConfig(),
			requestedPath: "/admin",
			expectedCode: 404,
			expectedBody: "no rule matched path: /admin",
		},
		{
			name: "Prefix path",
			cfg: validConfig(),
			requestedPath: "/api/v1/user",
			expectedCode: 200,
			expectedBody: "matched rule: /api/v1/*, upstreams: [node1 node2]",
		},
	}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			cfg := entry.cfg
			handler := NewHandler(cfg)

			// 1. Create a MOCK http request
			req := httptest.NewRequest(http.MethodGet, entry.requestedPath, nil)

			// 2. Create a response recorder to capture the handler's output
			res := httptest.NewRecorder()

			// 3. Invoke the handler function
			handler(res, req)

			if res.Code != entry.expectedCode {
				t.Errorf("expected %d, got %d", entry.expectedCode, res.Code)
			}

			if res.Body.String() != entry.expectedBody {
				t.Errorf("expected %s, got %s", entry.expectedBody, res.Body.String())
			}
		})
	}
}