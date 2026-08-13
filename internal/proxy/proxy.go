package proxy

import (
	"fmt"
	"net/http"
)


func Forward(upstreamURL string, r *http.Request) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, upstreamURL, r.Body)

	if err != nil {
		return nil, fmt.Errorf("failed create new request: %w", err)
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}

	response, err := client.Do(req)

	return response, err
}

func Relay(w http.ResponseWriter, resp *http.Response) {

}