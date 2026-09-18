package global

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/navire-dev/navire/core/internal/utils"
)

type RateLimitConfig struct {
	Burst             int
	RefillPerIPPerMin int
	MaxEntries        int
	SweepInterval     time.Duration
	IdleTTL           time.Duration
	TrustProxy        bool // NEW: resolve IP from proxy headers when true
	TrustedProxies    *utils.IPMatcher
}

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	lastRef  time.Time
	lastSeen time.Time
}

type limiter struct {
	cfg       RateLimitConfig
	rate      float64
	capacity  float64
	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

func newLimiter(cfg RateLimitConfig) *limiter {
	if cfg.SweepInterval <= 0 {
		cfg.SweepInterval = time.Minute
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 15 * time.Minute
	}
	if cfg.Burst < 1 {
		cfg.Burst = 1
	}
	if cfg.RefillPerIPPerMin < 1 {
		cfg.RefillPerIPPerMin = 1
	}
	return &limiter{
		cfg:       cfg,
		rate:      float64(cfg.RefillPerIPPerMin) / 60.0,
		capacity:  float64(cfg.Burst),
		buckets:   make(map[string]*bucket, 1024),
		lastSweep: time.Now(),
	}
}

func (l *limiter) getBucket(key string, now time.Time) (*bucket, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b == nil {
		if l.cfg.MaxEntries > 0 && len(l.buckets) >= l.cfg.MaxEntries {
			l.sweepLocked(now)
			if len(l.buckets) >= l.cfg.MaxEntries {
				return nil, false
			}
		}
		b = &bucket{tokens: l.capacity, lastRef: now, lastSeen: now}
		l.buckets[key] = b
	}
	return b, true
}

func (l *limiter) allow(key string, now time.Time) (ok bool, remaining int, retryAfterSec int) {
	b, available := l.getBucket(key, now)
	if !available {
		return false, 0, 1
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := now.Sub(b.lastRef).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(l.capacity, b.tokens+elapsed*l.rate)
		b.lastRef = now
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		b.lastSeen = now
		return true, int(math.Floor(b.tokens)), 0
	}

	needed := 1.0 - b.tokens
	sec := int(math.Ceil(needed / l.rate))
	if sec < 1 {
		sec = 1
	}
	return false, int(math.Floor(b.tokens)), sec
}

func (l *limiter) sweepLocked(now time.Time) {
	ttl := l.cfg.IdleTTL
	for ip, b := range l.buckets {
		b.mu.Lock()
		idle := now.Sub(b.lastSeen) > ttl
		b.mu.Unlock()
		if idle {
			delete(l.buckets, ip)
		}
	}
	l.lastSweep = now
}

func (l *limiter) sweepMaybe(now time.Time) {
	l.mu.Lock()
	if now.Sub(l.lastSweep) >= l.cfg.SweepInterval {
		l.sweepLocked(now)
	}
	l.mu.Unlock()
}

func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	l := newLimiter(cfg)
	limitStr := strconv.Itoa(l.cfg.Burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			l.sweepMaybe(now)

			key := utils.ClientIP(r, l.cfg.TrustProxy, l.cfg.TrustedProxies)

			ok, remaining, retry := l.allow(key, now)
			if !ok {
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				w.Header().Set("X-RateLimit-Limit", limitStr)
				w.Header().Set("X-RateLimit-Remaining", "0")
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			// Write informational headers AFTER the handler, so they reflect this request's consumption.
			defer func(rem int) {
				if rem < 0 {
					rem = 0
				}
				w.Header().Set("X-RateLimit-Limit", limitStr)
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rem))
			}(remaining)

			next.ServeHTTP(w, r)
		})
	}
}
