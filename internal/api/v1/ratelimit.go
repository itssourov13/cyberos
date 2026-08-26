package v1

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// RateLimitConfig bounds AI/autonomy API abuse (process-local).
// Zero Limit uses DefaultAIRateLimit / DefaultAutonomyRateLimit.
type RateLimitConfig struct {
	AILimit       int           // requests per window for /api/v1/ai/*
	AutonomyLimit int           // requests per window for /api/v1/autonomy/*
	Window        time.Duration // fixed window
	MaxBodyBytes  int64         // request body cap
}

// Defaults — fail-closed bounded values (not unlimited).
const (
	DefaultAIRateLimit       = 60
	DefaultAutonomyRateLimit = 60
	DefaultRateWindow        = time.Minute
	DefaultMaxBodyBytes      = 1 << 20 // 1 MiB
)

func (c RateLimitConfig) normalized() RateLimitConfig {
	if c.AILimit <= 0 {
		c.AILimit = DefaultAIRateLimit
	}
	if c.AutonomyLimit <= 0 {
		c.AutonomyLimit = DefaultAutonomyRateLimit
	}
	if c.Window <= 0 {
		c.Window = DefaultRateWindow
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = DefaultMaxBodyBytes
	}
	return c
}

// fixedWindowLimiter is concurrency-safe, process-local.
type fixedWindowLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	buckets map[string]*windowBucket
	now     func() time.Time
}

type windowBucket struct {
	windowStart time.Time
	count       int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		window: window, limit: limit, buckets: make(map[string]*windowBucket), now: time.Now,
	}
}

// allow returns whether the request may proceed and seconds until retry if not.
func (l *fixedWindowLimiter) allow(key string) (ok bool, retryAfterSec int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, exists := l.buckets[key]
	if !exists || now.Sub(b.windowStart) >= l.window {
		l.buckets[key] = &windowBucket{windowStart: now, count: 1}
		return true, 0
	}
	if b.count >= l.limit {
		rem := l.window - now.Sub(b.windowStart)
		sec := int(rem.Seconds())
		if sec < 1 {
			sec = 1
		}
		return false, sec
	}
	b.count++
	return true, 0
}

// SafetyLimiter holds AI and autonomy rate limiters.
type SafetyLimiter struct {
	AI       *fixedWindowLimiter
	Autonomy *fixedWindowLimiter
	Cfg      RateLimitConfig
}

func NewSafetyLimiter(cfg RateLimitConfig) *SafetyLimiter {
	cfg = cfg.normalized()
	return &SafetyLimiter{
		AI:       newFixedWindowLimiter(cfg.AILimit, cfg.Window),
		Autonomy: newFixedWindowLimiter(cfg.AutonomyLimit, cfg.Window),
		Cfg:      cfg,
	}
}

func rateLimitKey(r *http.Request, group string) string {
	prin := ""
	if p, ok := auth.PrincipalFromContext(r.Context()); ok {
		prin = string(p.ID)
	}
	proj := r.Header.Get("X-CyberOS-Project")
	if proj == "" {
		proj = r.URL.Query().Get("project_id")
	}
	return group + "|" + prin + "|" + proj
}

// limitRequest applies fixed-window limit; writes 429 on exhaustion.
func (s *Server) limitRequest(w http.ResponseWriter, r *http.Request, group string) bool {
	if s.Limiter == nil {
		return true
	}
	var lim *fixedWindowLimiter
	switch group {
	case "ai":
		lim = s.Limiter.AI
	default:
		lim = s.Limiter.Autonomy
	}
	ok, retry := lim.allow(rateLimitKey(r, group))
	if ok {
		return true
	}
	w.Header().Set("Retry-After", strconv.Itoa(retry))
	writeErr(w, http.StatusTooManyRequests, "rate limit exceeded")
	return false
}

// limitBody wraps r.Body with MaxBytesReader.
func (s *Server) limitBody(w http.ResponseWriter, r *http.Request) {
	max := int64(DefaultMaxBodyBytes)
	if s.Limiter != nil && s.Limiter.Cfg.MaxBodyBytes > 0 {
		max = s.Limiter.Cfg.MaxBodyBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, max)
}

// silence unused
var _ = contracts.ProjectID("")
