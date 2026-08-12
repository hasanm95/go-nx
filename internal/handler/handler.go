package handler

import (
	"fmt"
	"net/http"

	"github.com/hasanm95/go-nx/internal/config"
	"github.com/hasanm95/go-nx/internal/matcher"
)

func NewHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestePath := r.URL.Path
		match, ok := matcher.MatchPath(cfg.Server.Paths, requestePath)

		if !ok {
			responseString := fmt.Sprintf("no rule matched path: %s", requestePath)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(responseString))
		} else {
			responseString := fmt.Sprintf("matched rule: %s, upstreams: %v", match.Path, match.Upstreams)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(responseString))
		}
	}
}