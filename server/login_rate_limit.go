package server

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// loginRateLimiter throttles login failures per client IP
// (specs/001-login-rate-limit). Single mutex + map is deliberate: the login
// path is not hot (bcrypt costs ~80ms/call) and map ops are orders of
// magnitude cheaper; sharding would be unjustified complexity.
//
// State machine (per IP):
//
//	Empty --failure--> Active --failure#5--> Blocked --cooldown elapsed--> Empty
//	Active --success--> Empty (Reset)
//	Active --window expired--> Active (counting restarts at the next failure)
type loginRateLimiter struct {
	mu     sync.Mutex
	states map[string]*loginState
}

type loginState struct {
	failures     int       // failures inside the current window
	windowStart  time.Time // start of the current counting window
	blockedUntil time.Time // zero = never blocked
}

const (
	loginMaxFailures = 5
	loginWindow      = 5 * time.Minute
	loginCooldown    = 5 * time.Minute
	loginGCInterval  = 10 * time.Minute
)

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{states: map[string]*loginState{}}
}

// Check reports whether ip is currently blocked (FR-002) and, if so, how long
// remains of the cooldown. Expired cooldowns / windows are lazily cleared so
// the next failure starts counting from one (AC3).
func (lm *loginRateLimiter) Check(ip string) (blocked bool, retryAfter time.Duration) {
	now := time.Now()
	lm.mu.Lock()
	defer lm.mu.Unlock()
	st, ok := lm.states[ip]
	if !ok {
		return false, 0
	}
	if st.blockedUntil.After(now) {
		return true, st.blockedUntil.Sub(now)
	}
	// Cooldown elapsed or window expired: drop the entry so the IP starts
	// fresh — this is what makes recovery automatic (AC3).
	if !st.blockedUntil.IsZero() || now.Sub(st.windowStart) > loginWindow {
		delete(lm.states, ip)
	}
	return false, 0
}

// RecordFailure counts one failed login for ip and returns true only on the
// transition into the blocked state (loginMaxFailures reached), which the
// caller uses to emit the block log line (FR-010). Failures past the
// threshold keep counting but never re-extend the cooldown.
func (lm *loginRateLimiter) RecordFailure(ip string) (blocked bool) {
	now := time.Now()
	lm.mu.Lock()
	defer lm.mu.Unlock()
	st, ok := lm.states[ip]
	if !ok {
		st = &loginState{windowStart: now}
		lm.states[ip] = st
	}
	if now.Sub(st.windowStart) > loginWindow {
		st.failures = 0
		st.windowStart = now
	}
	st.failures++
	if st.failures == loginMaxFailures {
		st.blockedUntil = now.Add(loginCooldown)
		return true
	}
	return false
}

// Reset clears all state for ip after a successful login (FR-004).
func (lm *loginRateLimiter) Reset(ip string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.states, ip)
}

// gcOnce removes entries whose counting window has expired AND whose cooldown
// (if any) has elapsed, keeping memory bounded under IP-spoofing pressure
// (FR-009). Still-blocked entries always survive.
func (lm *loginRateLimiter) gcOnce() {
	now := time.Now()
	lm.mu.Lock()
	defer lm.mu.Unlock()
	for ip, st := range lm.states {
		windowExpired := now.Sub(st.windowStart) > loginWindow
		cooldownElapsed := st.blockedUntil.IsZero() || !st.blockedUntil.After(now)
		if windowExpired && cooldownElapsed {
			delete(lm.states, ip)
		}
	}
}

// startGC runs gcOnce periodically until ctx is canceled (T014).
func (lm *loginRateLimiter) startGC(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(loginGCInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lm.gcOnce()
			}
		}
	}()
}

// extractClientIP resolves the client IP for rate limiting (US2 / FR-006,
// FR-007). X-Forwarded-For is honored ONLY when trustProxy is true — spoofed
// headers must not bypass the limiter — and then only its first hop.
func extractClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// writeLoginRateLimited emits the 429 response with an RFC 9110 §10.7.3
// Retry-After header and a retry_after body field (FR-002 / AC7). The error
// message deliberately stays vague — it must not reveal how many attempts
// remain before the block.
func writeLoginRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":       "登录失败次数过多，请稍后再试",
		"retry_after": secs,
	})
}

// logBlocked emits the single INFO line allowed per block (FR-010 / spec A4).
// Ordinary (below-threshold) failures are NOT logged — a flood of failures
// must not become a disk-filling log-flood vector.
func logBlocked(ip string, failures int) {
	log.Printf("[auth] IP=%s blocked after %d failures, cooldown=%ds", ip, failures, int(loginCooldown.Seconds()))
}
