package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestMaxBodyMiddlewareRejectsLargePayloads(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := maxBodyMiddleware(10)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this payload is longer than ten bytes"))
	req.Header.Set("Content-Length", "40")
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rr.Code)
	}
}

func TestIdempotencyMiddlewareRejectsDuplicateKeys(t *testing.T) {
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})
	mw := idempotencyMiddleware(next)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Idempotency-Key", "abc-123")
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}
	if called != 1 {
		t.Fatalf("expected handler to be called once for duplicate idempotency key, got %d", called)
	}
}
