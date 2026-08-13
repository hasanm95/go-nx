package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)



func TestForward_GETSuccess(t *testing.T){
	type Resp struct {
		Status  string `json:"status"`
	}
	// 1. Create a MOCK server
	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		if r.URL.Path != "/users" {
			t.Errorf("expected path /users, got %s", r.URL.Path)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "bearer 12345" {
			t.Errorf("expected 'Authorization' header 'bearer 12345', got %s", authHeader)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer mockServer.Close() 

	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Add("Authorization", "bearer 12345")

	targetURL := mockServer.URL + "/users"
	response, err := Forward(targetURL, req)

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

	targetURL := mockServer.URL + "/users"
	Forward(targetURL, req)
}

func TestForward_NetworkFailure(t *testing.T){
	// 1. Create a MOCK server
	mockServer := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {

	}))
	defer mockServer.Close() 
	
	req := httptest.NewRequest("GET", "/users", nil)
	invalidURL := "http://this-domain-does-not-exist-at-all-12345.xyz"
	response, err := Forward(invalidURL, req)

	if err == nil {
		t.Fatal("expected an error due to network failure, but got nil")
	}

	if response != nil {
		t.Errorf("expected response to be nil on failure, got %v", response)
	}
}