package proxy

import (
	"net/http/httptest"
	"testing"

	"github.com/hasanm95/go-nx/internal/config"
)

func TestSubstituteVariable(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.5:1234"

	if got := SubstituteVariable("$ip", req); got != "10.0.0.5" {
		t.Errorf("expected 10.0.0.5, got %s", got)
	}

	if got := SubstituteVariable("static-value", req); got != "static-value" {
		t.Errorf("expected unchanged literal, got %s", got)
	}
}

func TestApplyConfiguredHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	original := httptest.NewRequest("GET", "/", nil)
	original.RemoteAddr = "10.0.0.5:1234"

	headers := []config.HeaderConfig{
		{Key: "X-Forwarded-For", Value: "$ip"},
		{Key: "X-Static", Value: "hello"},
	}

	ApplyConfiguredHeaders(req, headers, original)

	if got := req.Header.Get("X-Forwarded-For"); got != "10.0.0.5" {
		t.Errorf("expected 10.0.0.5, got %s", got)
	}
	if got := req.Header.Get("X-Static"); got != "hello" {
		t.Errorf("expected hello, got %s", got)
	}
}