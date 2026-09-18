package global

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitAllowsConfiguredBurst(t *testing.T) {
	handler := RateLimit(RateLimitConfig{
		Burst:             2,
		RefillPerIPPerMin: 1,
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/events", http.NoBody)
		request.RemoteAddr = "192.0.2.1:1234"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, recorder.Code, http.StatusOK)
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", http.NoBody)
	request.RemoteAddr = "192.0.2.1:1234"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("third request status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitCapsBucketEntries(t *testing.T) {
	limiter := newLimiter(RateLimitConfig{
		Burst:             1,
		RefillPerIPPerMin: 1,
		MaxEntries:        1,
		IdleTTL:           time.Hour,
	})

	if ok, _, _ := limiter.allow("192.0.2.1", time.Now()); !ok {
		t.Fatal("first client was rejected")
	}
	if ok, _, _ := limiter.allow("192.0.2.2", time.Now()); ok {
		t.Fatal("second client was accepted after the bucket limit was reached")
	}
	if got := len(limiter.buckets); got != 1 {
		t.Fatalf("bucket count = %d, want 1", got)
	}
}
