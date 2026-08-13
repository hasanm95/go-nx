package proxy

import (
	"fmt"
	"io"
	"net/http"
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

func (p *Proxy) Forward(upstreamURL string, r *http.Request) (*http.Response, error) {

	targetURL := upstreamURL + r.URL.Path 

	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequest(r.Method, targetURL, r.Body)

	if err != nil {
		return nil, fmt.Errorf("failed create new request: %w", err)
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	response, err := p.Client.Do(req)

	return response, err
}

func Relay(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}