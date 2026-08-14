package proxy

import (
	"net"
	"net/http"

	"github.com/hasanm95/go-nx/internal/config"
)

type VariableResolver func (r *http.Request) string

var variableResolvers = map[string]VariableResolver {
	"$ip": func(r *http.Request) string {
		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)

		if err != nil {
			return r.RemoteAddr
		}

		return clientIP
	},
}

func SubstituteVariable(value string, r *http.Request) string {
	if resolver, ok := variableResolvers[value]; ok {
		return resolver(r)
	}
	return value
}

func ApplyConfiguredHeaders(req *http.Request, headers []config.HeaderConfig, original *http.Request) {
	for _, h := range headers {
		req.Header.Set(h.Key, SubstituteVariable(h.Value, original))
	}
}