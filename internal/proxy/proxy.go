package proxy

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

type Proxy struct {
	Client *http.Client
}

func NewProxy() *Proxy {
	return &Proxy{
		Client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns: 100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

func copyHeaders(dst, src http.Header) {
	connHeader := src.Get("Connection")

	connHeaders := strings.Split(connHeader, ",")
	for i := range connHeaders {
		connHeaders[i] = strings.ToLower(strings.TrimSpace(connHeaders[i]))
	}

	connHeaders = slices.DeleteFunc(connHeaders, func(s string) bool {
		return s == ""
	})

	for key, values := range src {
		lowerKey := strings.ToLower(key)

		if _, exists := hopByHopHeaders[lowerKey]; exists {
			continue
		}

		if slices.Contains(connHeaders, lowerKey) {
			continue
		}

		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func (p *Proxy) Forward(upstreamURL string, r *http.Request) (*http.Response, error) {
	targetURL := upstreamURL + r.URL.Path

	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		targetURL,
		r.Body,
	)

	if err != nil {
		return nil, fmt.Errorf("failed create new request: %w", err)
	}

	copyHeaders(req.Header, r.Header)
	req.Host = r.Host

	response, err := p.Client.Do(req)

	return response, err
}

func (p *Proxy) Relay(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}