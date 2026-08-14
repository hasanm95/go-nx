package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hasanm95/go-nx/internal/config"
)

type Proxy struct {
	Client *http.Client
}

func NewProxy() *Proxy {
	return &Proxy{
		Client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
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

func copyFilteredHeaders(dst, src http.Header) {
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

func addForwardedHeaders(req *http.Request, original *http.Request) {
	clientIP, _, err := net.SplitHostPort(original.RemoteAddr)
	if err != nil {
		clientIP = original.RemoteAddr
	}

	existing := original.Header.Get("X-Forwarded-For")

	if existing != "" {
		req.Header.Set("X-Forwarded-For", existing+", "+clientIP)
	} else {
		req.Header.Set("X-Forwarded-For", clientIP)
	}

	proto := "http"
	if original.TLS != nil {
		proto = "https"
	}

	req.Header.Set("X-Forwarded-Proto", proto)
	req.Header.Set("X-Forwarded-Host", original.Host)
}

func (p *Proxy) Forward(upstreamURL string, r *http.Request, headers []config.HeaderConfig) (*http.Response, error) {
	targetURL := strings.TrimSuffix(upstreamURL, "/") + r.URL.Path

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

	req.ContentLength = r.ContentLength

	copyFilteredHeaders(req.Header, r.Header)
	req.Host = r.Host
	addForwardedHeaders(req, r)
	ApplyConfiguredHeaders(req, headers, r)

	return p.Client.Do(req)
}

func (p *Proxy) Relay(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()

	copyFilteredHeaders(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}