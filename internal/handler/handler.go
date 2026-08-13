package handler

import (
	"fmt"
	"net/http"

	"github.com/hasanm95/go-nx/internal/config"
	"github.com/hasanm95/go-nx/internal/matcher"
	"github.com/hasanm95/go-nx/internal/proxy"
	"github.com/hasanm95/go-nx/internal/upstream"
)

func NewHandler(cfg *config.Config, selectors map[string]func() string, p *proxy.Proxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedPath := r.URL.Path
		match, ok := matcher.MatchPath(cfg.Server.Paths, requestedPath)

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "no rule matched path: %s", requestedPath)
			return
		}

		selectNext, ok := selectors[match.Path]
		if !ok {
			http.Error(w, fmt.Sprintf("no selector configured for rule: %s", match.Path), http.StatusInternalServerError)
			return
		}

		selectedID := selectNext()
		upstreamURL, ok := upstream.Lookup(selectedID, cfg.Server.Upstreams)
		if !ok {
			http.Error(w, fmt.Sprintf("no upstream found for id: %s", selectedID), http.StatusInternalServerError)
			return
		}

		resp, err := p.Forward(upstreamURL, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to reach upstream: %v", err), http.StatusBadGateway)
			return
		}

		p.Relay(w, resp)
	}
}