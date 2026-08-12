package upstream

import (
	"sync/atomic"

	"github.com/hasanm95/go-nx/internal/config"
)

func Selector(upstreams []string) func() string {
	if len(upstreams) == 0 {
		return func() string {
			return ""
		}
	}

	var counter uint64
	return func() string {
		idx := atomic.AddUint64(&counter, 1) - 1
		return upstreams[idx%uint64(len(upstreams))]
	}
}

func Lookup(selected string, Upstreams []config.UpstreamConfig) (value string, ok bool) {
	for _, upstream := range Upstreams {
		if upstream.ID == selected {
			return upstream.URL, true
		}
	}
	return "", false
}