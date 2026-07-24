package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	cfg := ServerConfig{APIKey: "super-secret", EnableAuth: true}
	req := httptest.NewRequest(http.MethodGet, "/api/chain", nil)
	if err := requireAuth(req, cfg); err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
}

func TestRequireAuthAcceptsBearerToken(t *testing.T) {
	cfg := ServerConfig{APIKey: "super-secret"}
	req := httptest.NewRequest(http.MethodGet, "/api/chain", nil)
	req.Header.Set("Authorization", "Bearer super-secret")
	if err := requireAuth(req, cfg); err != nil {
		t.Fatalf("expected valid bearer token to pass, got %v", err)
	}
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := newRateLimiter(2, time.Second)
	if !limiter.allow("client-a") {
		t.Fatal("expected first request to be allowed")
	}
	if !limiter.allow("client-a") {
		t.Fatal("expected second request to be allowed")
	}
	if limiter.allow("client-a") {
		t.Fatal("expected third request to be rejected")
	}
}
