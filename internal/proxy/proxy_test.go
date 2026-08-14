package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)



func TestForward_GETSuccess(t *testing.T){
	type Resp struct {
		Status  string `json:"status"`
	}
	proxy := NewProxy()

	// 1. Create a MOCK server
	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		if r.URL.Path != "/users" {
			t.Errorf("expected path /users, got %s", r.URL.Path)
		}

		pageQuery := r.URL.Query().Get("page") 
		if pageQuery != "2" {
			t.Errorf("expected 'page' query 2, got %s", pageQuery)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "bearer 12345" {
			t.Errorf("expected 'Authorization' header 'bearer 12345', got %s", authHeader)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer mockServer.Close() 

	req := httptest.NewRequest("GET", "/users?page=2", nil)
	req.Header.Add("Authorization", "bearer 12345")

	response, err := proxy.Forward(mockServer.URL, req, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer response.Body.Close() 

	if response.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", response.StatusCode)
	}

	var resp Resp
	err = json.NewDecoder(response.Body).Decode(&resp)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success, got %s", resp.Status)
	}
}

func TestForward_POSTSuccess(t *testing.T){
	proxy := NewProxy()

	// 1. Create a MOCK server
	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {

		if r.Method != "POST" {
			t.Errorf("expected 'POST', got %s", r.Method)
		}

		bodyBytes, _ := io.ReadAll(r.Body)

		if string(bodyBytes) != `{"name":"gopher"}` {
			t.Errorf("expected body to be forwarded, got %s", string(bodyBytes))
		}
	}))
	defer mockServer.Close() 

	bodyReader := bytes.NewReader([]byte(`{"name":"gopher"}`))
	req := httptest.NewRequest("POST", "/users", bodyReader)
	req.Header.Add("Content-Type", "application/json")

	proxy.Forward(mockServer.URL, req, nil)
}

func TestForward_NetworkFailure(t *testing.T){
	proxy := NewProxy()

	// 1. Create a MOCK server
	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {

	}))
	defer mockServer.Close() 
	
	req := httptest.NewRequest("GET", "/users", nil)
	invalidURL := "http://this-domain-does-not-exist-at-all-12345.xyz"
	response, err := proxy.Forward(invalidURL, req, nil)

	if err == nil {
		t.Fatal("expected an error due to network failure, but got nil")
	}

	if response != nil {
		t.Errorf("expected response to be nil on failure, got %v", response)
	}
}

func TestForward_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	proxy := NewProxy()

	requestReceived := make(chan struct{})

	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		reqCtx := r.Context()
		close(requestReceived)

		<- reqCtx.Done()
	}))
	defer mockServer.Close() 

	req := httptest.NewRequestWithContext(ctx, "GET", "/users", nil)

	type ProxyResult struct {
		res *http.Response
		err error
	}

	ch := make(chan ProxyResult, 1)

	go func() {
		res, err := proxy.Forward(mockServer.URL, req, nil)

		ch <- ProxyResult{res: res, err: err}
	}()

	<-requestReceived
	cancel()

	result := <-ch

	if result.err == nil {
		t.Errorf("expected context canceled error, got none")
	}

	if result.err != nil {
		if !errors.Is(result.err, context.Canceled){
			t.Errorf("expected context canceled error, got %v", result.err)
		}
	}
}

func TestForward_HopByHopHeaders(t *testing.T){
	proxy := NewProxy()

	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		if r.Header.Get("Connection") != "" {
			t.Errorf("expected Connection header to be removed")
		}

		if r.Header.Get("X-Internal") != "" {
			t.Errorf("expected X-Internal to be removed")
		}

		if r.Header.Get("X-Test-Header") != "hello" {
			t.Errorf("expected X-Test-Header to be forwarded")
		}

		if r.Host != "example.com" {
			t.Errorf("expected Host 'example.com', got %q", r.Host)
		}
	}))
	defer mockServer.Close() 

	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Add("Connection", "keep-alive, X-Internal")
	req.Header.Add("X-Test-Header", "hello")
	req.Header.Add("X-Internal", "secret")
	req.Host = "example.com"

	_, _ = proxy.Forward(mockServer.URL, req, nil)
}

func TestForward_XForwardedFor(t *testing.T) {
	proxy := NewProxy()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Forwarded-For")

		if got != "192.168.1.20" {
			t.Errorf("expected X-Forwarded-For '192.168.1.20', got %q", got)
		}
	}))
	defer mockServer.Close()

	req := httptest.NewRequest("GET", "/users", nil)
	req.RemoteAddr = "192.168.1.20:54321"

	_, err := proxy.Forward(mockServer.URL, req, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestForward_XForwardedFor_Append(t *testing.T) {
	proxy := NewProxy()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Forwarded-For")

		expected := "10.0.0.5, 192.168.1.20"

		if got != expected {
			t.Errorf("expected X-Forwarded-For %q, got %q", expected, got)
		}
	}))
	defer mockServer.Close()

	req := httptest.NewRequest("GET", "/users", nil)
	req.RemoteAddr = "192.168.1.20:54321"
	req.Header.Set("X-Forwarded-For", "10.0.0.5")

	_, err := proxy.Forward(mockServer.URL, req, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestForward_XForwardedProto_HTTP(t *testing.T) {
	proxy := NewProxy()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Forwarded-Proto")

		if got != "http" {
			t.Errorf("expected X-Forwarded-Proto 'http', got %q", got)
		}
	}))
	defer mockServer.Close()

	req := httptest.NewRequest("GET", "/users", nil)

	_, err := proxy.Forward(mockServer.URL, req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestForward_XForwardedProto_HTTPS(t *testing.T) {
	proxy := NewProxy()

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Forwarded-Proto")

		if got != "https" {
			t.Errorf("expected X-Forwarded-Proto 'https', got %q", got)
		}
	}))
	defer mockUpstream.Close()

	req := httptest.NewRequest("GET", "https://example.com/users", nil)
	req.TLS = &tls.ConnectionState{}

	_, err := proxy.Forward(mockUpstream.URL, req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestForward_XForwardedHost(t *testing.T) {
	proxy := NewProxy()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Forwarded-Host")

		if got != "example.com" {
			t.Errorf("expected X-Forwarded-Host 'example.com', got %q", got)
		}
	}))
	defer mockServer.Close()

	req := httptest.NewRequest("GET", "/users", nil)
	req.Host = "example.com"

	_, err := proxy.Forward(mockServer.URL, req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRelay_Success(t *testing.T) {
	proxy := NewProxy()

	// Create MOCK response writer
	rec := httptest.NewRecorder()

	// Create MOCK response
	bodyBytes := []byte(`{"status":"success"}`)
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Header: make(http.Header),
		Body: io.NopCloser(bytes.NewBuffer(bodyBytes)),
	}
	mockResp.Header.Set("Content-Type", "application/json")

	// Call Relay
	proxy.Relay(rec, mockResp)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected %d, go %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	if rec.Body.String() != `{"status":"success"}` {
		t.Errorf("expected body '{\"status\":\"success\"}', got '%s'", rec.Body.String())
	}
}
func TestRelay_HopByHopHeaders(t *testing.T) {
	proxy := NewProxy()
	rec := httptest.NewRecorder()

	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
	}

	mockResp.Header.Set("Connection", "keep-alive")
	mockResp.Header.Set("X-Test-Header", "hello")

	proxy.Relay(rec, mockResp)

	if got := rec.Header().Get("Connection"); got != "" {
		t.Errorf("expected Connection header to be removed, got %q", got)
	}

	if got := rec.Header().Get("X-Test-Header"); got != "hello" {
		t.Errorf("expected X-Test-Header to be forwarded, got %q", got)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != `{"status":"success"}` {
		t.Errorf("expected body %q, got %q", `{"status":"success"}`, rec.Body.String())
	}
}