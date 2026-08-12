package handler

import (
	"fmt"
	"net/http"

	"github.com/hasanm95/go-nx/internal/config"
	"github.com/hasanm95/go-nx/internal/matcher"
	"github.com/hasanm95/go-nx/internal/upstream"
)

func NewHandler(cfg *config.Config, selectors map[string]func() string) http.HandlerFunc {
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
			// Shouldn't happen if selectors were built from this same cfg.Server.Paths —
			// guarded rather than assumed, so a mismatch fails loudly instead of panicking.
			http.Error(w, fmt.Sprintf("no selector configured for rule: %s", match.Path), http.StatusInternalServerError)
			return
		}

		selectedID := selectNext()
		upstreamURL, ok := upstream.Lookup(selectedID, cfg.Server.Upstreams)

		if !ok && upstreamURL == "" {
			http.Error(w, fmt.Sprintf("no upstream found for id: %s", selectedID), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "matched rule: %s, selected upstream: %s -> %s", match.Path, selectedID, upstreamURL)
	}
}