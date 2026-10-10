package server

// Tests for the login rate limiter (specs/001-login-rate-limit).
// Traceability: test names map to spec.md user stories / acceptance criteria:
//   TestLoginRateStateMachine  → US1 (AC1–AC5 state machine behavior)
//   TestExtractClientIP        → US2 (AC9)
//   TestLoginRateConcurrent    → US3 (AC10)
//   TestLoginRateGC            → US3 (AC11)
//   TestAuthLoginRateLimit     → US1+US2 integration through the real mux (AC1, AC3, AC4, AC6, AC7, AC8)

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const testRateLimitIP = "203.0.113.7"

// newTestLoginServer builds a Server backed by real postgres (skipped when
// unavailable) and provisions a KNOWN admin password without ever hardcoding
// one in source:
//  1. fresh DB (e.g. the CI service container): POST /api/auth/init with a
//     random in-memory password — the production first-run flow itself;
//  2. already-initialized DB (local dev): the ARTEX_TEST_LOGIN_PASSWORD env
//     var, verified with a probe login — the real credential stays out of the
//     repo;
//  3. otherwise the integration test skips.
func newTestLoginServer(t *testing.T) (*Server, http.Handler, string) {
	t.Helper()
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { m.Close() })
	td := t.TempDir()
	s := New(context.Background(), m, td, td, td)
	h := s.Handler()

	pwd := randTestPassword()
	if rec := postInit(h, pwd); rec.Code == http.StatusOK {
		return s, h, pwd
	}
	if envPwd := os.Getenv("ARTEX_TEST_LOGIN_PASSWORD"); envPwd != "" {
		if rec := postLogin(h, "10.255.255.254", envPwd); rec.Code == http.StatusOK {
			return s, h, envPwd
		}
		t.Skipf("ARTEX_TEST_LOGIN_PASSWORD does not match the dev DB admin password — skipping")
	}
	t.Skip("admin password already initialized and ARTEX_TEST_LOGIN_PASSWORD unset — skipping integration test")
	return nil, nil, ""
}

// randTestPassword returns a random 16-hex-char password; it exists only in
// process memory for the duration of the test.
func randTestPassword() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err) // crypto/rand failure is unrecoverable in tests
	}
	return hex.EncodeToString(buf[:])
}

// postInit drives the production first-run endpoint to claim the password.
func postInit(h http.Handler, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"password": password})
	req := httptest.NewRequest("POST", "/api/auth/init", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// postLogin sends a login request pretending to originate from ip.
func postLogin(h http.Handler, ip string, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"username": "ARTEX", "password": password})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.RemoteAddr = ip + ":44444"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestLoginRateStateMachine covers the per-IP state machine transitions from
// plan.md §数据模型: Empty → Active → Blocked → (cooldown expired) → Empty,
// plus success-Reset and window-expiry reset.
func TestLoginRateStateMachine(t *testing.T) {
	lm := newLoginRateLimiter()

	// Empty: unknown IP is never blocked.
	if blocked, _ := lm.Check("1.1.1.1"); blocked {
		t.Fatal("unknown IP must not be blocked")
	}

	// Active: 4 failures stay below the threshold (AC5 boundary).
	for i := 0; i < loginMaxFailures-1; i++ {
		if b := lm.RecordFailure("1.1.1.1"); b {
			t.Fatalf("failure %d must not trigger block", i+1)
		}
	}
	if blocked, _ := lm.Check("1.1.1.1"); blocked {
		t.Fatal("4 failures must not block yet (AC5)")
	}
	if got := lm.states["1.1.1.1"].failures; got != loginMaxFailures-1 {
		t.Fatalf("failures=%d want %d", got, loginMaxFailures-1)
	}

	// Active → Blocked: the 5th failure triggers the block (AC1).
	if !lm.RecordFailure("1.1.1.1") {
		t.Fatal("5th failure must return blocked=true (transition signal)")
	}
	blocked, retryAfter := lm.Check("1.1.1.1")
	if !blocked {
		t.Fatal("5 failures must block (AC1)")
	}
	if retryAfter <= 0 || retryAfter > loginCooldown {
		t.Fatalf("retryAfter=%v want in (0,%v]", retryAfter, loginCooldown)
	}

	// Check is read-only: repeated checks keep the state blocked and the
	// failure count untouched.
	for i := 0; i < 3; i++ {
		if b, _ := lm.Check("1.1.1.1"); !b {
			t.Fatal("must remain blocked within cooldown")
		}
	}
	if got := lm.states["1.1.1.1"].failures; got != loginMaxFailures {
		t.Fatalf("Check must not mutate failures: got %d want %d", got, loginMaxFailures)
	}

	// Cooldown expired → Check unblocks and clears the entry (AC3).
	lm.mu.Lock()
	lm.states["1.1.1.1"].blockedUntil = time.Now().Add(-time.Second)
	lm.mu.Unlock()
	if blocked, _ := lm.Check("1.1.1.1"); blocked {
		t.Fatal("expired cooldown must unblock (AC3)")
	}
	if _, ok := lm.states["1.1.1.1"]; ok {
		t.Fatal("expired entry must be cleared so next failure starts at 1")
	}

	// Window expiry: an Active state older than the window resets counting.
	lm2 := newLoginRateLimiter()
	for i := 0; i < 3; i++ {
		lm2.RecordFailure("2.2.2.2")
	}
	lm2.mu.Lock()
	lm2.states["2.2.2.2"].windowStart = time.Now().Add(-loginWindow - time.Minute)
	lm2.mu.Unlock()
	lm2.RecordFailure("2.2.2.2") // window expired → reset then count this one
	if got := lm2.states["2.2.2.2"].failures; got != 1 {
		t.Fatalf("after window expiry failures=%d want 1", got)
	}

	// Reset on success (AC4).
	lm2.Reset("2.2.2.2")
	if _, ok := lm2.states["2.2.2.2"]; ok {
		t.Fatal("Reset must remove the entry")
	}
	if blocked, _ := lm2.Check("2.2.2.2"); blocked {
		t.Fatal("after Reset the IP must not be blocked")
	}
}

// TestExtractClientIP covers the IP extraction policy (US2 / AC9):
// trustProxy=false ignores X-Forwarded-For entirely; trustProxy=true takes
// the first hop; malformed/missing inputs fall back to RemoteAddr.
func TestExtractClientIP(t *testing.T) {
	cases := []struct {
		name       string
		trustProxy bool
		xff        string
		remote     string
		want       string
	}{
		{"no-trust ignores forged xff", false, "1.2.3.4", "9.9.9.9:1111", "9.9.9.9"},
		{"no-trust plain remote", false, "", "9.9.9.9:1111", "9.9.9.9"},
		{"trust takes first hop", true, "1.2.3.4, 5.6.7.8", "9.9.9.9:1111", "1.2.3.4"},
		{"trust empty xff falls back", true, "", "9.9.9.9:1111", "9.9.9.9"},
		{"trust blank xff falls back", true, "   ", "9.9.9.9:1111", "9.9.9.9"},
		{"ipv6 without port", false, "", "2001:db8::1", "2001:db8::1"},
		{"ipv6 with port", false, "", "[2001:db8::1]:443", "2001:db8::1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/login", nil)
			req.RemoteAddr = tc.remote
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := extractClientIP(req, tc.trustProxy); got != tc.want {
				t.Fatalf("extractClientIP=%q want %q", got, tc.want)
			}
		})
	}
}

// TestLoginRateConcurrent hammers the limiter from many goroutines; run under
// -race it proves the mutex discipline (US3 / AC10 / FR-008).
func TestLoginRateConcurrent(t *testing.T) {
	lm := newLoginRateLimiter()

	const n = 100
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			lm.RecordFailure(testRateLimitIP)
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	if got := lm.states[testRateLimitIP].failures; got != n {
		t.Fatalf("concurrent RecordFailure lost updates: failures=%d want %d", got, n)
	}

	// Blocked state is stable under concurrent checks.
	const checkers = 20
	res := make(chan bool, checkers)
	for i := 0; i < checkers; i++ {
		go func() { b, _ := lm.Check(testRateLimitIP); res <- b }()
	}
	for i := 0; i < checkers; i++ {
		if !<-res {
			t.Fatal("concurrent Check within cooldown must report blocked")
		}
	}
}

// TestLoginRateGC proves gcOnce removes expired unblocked entries and keeps
// entries still inside their window or still blocked (US3 / AC11 / FR-009).
func TestLoginRateGC(t *testing.T) {
	lm := newLoginRateLimiter()
	now := time.Now()

	lm.states["expired-free"] = &loginState{failures: 2, windowStart: now.Add(-2 * loginWindow)}
	lm.states["expired-blocked"] = &loginState{failures: 5, windowStart: now.Add(-2 * loginWindow), blockedUntil: now.Add(time.Minute)}
	lm.states["fresh"] = &loginState{failures: 1, windowStart: now}
	lm.states["cooldown-past"] = &loginState{failures: 5, windowStart: now.Add(-2 * loginWindow), blockedUntil: now.Add(-time.Second)}

	lm.gcOnce()

	if _, ok := lm.states["expired-free"]; ok {
		t.Fatal("expired unblocked entry must be GC'd")
	}
	if _, ok := lm.states["expired-blocked"]; !ok {
		t.Fatal("still-blocked entry must survive GC")
	}
	if _, ok := lm.states["fresh"]; !ok {
		t.Fatal("fresh entry must survive GC")
	}
	if _, ok := lm.states["cooldown-past"]; ok {
		t.Fatal("entry with elapsed cooldown must be GC'd")
	}
}

// TestAuthLoginRateLimit drives the full mux: real requireAuth exemptions,
// real bcrypt, real settings store — exercising AC1/AC3/AC4/AC6/AC7/AC8.
func TestAuthLoginRateLimit(t *testing.T) {
	s, h, loginPassword := newTestLoginServer(t)

	const (
		ipA = "198.51.100.10"
		ipB = "198.51.100.11"
		ipC = "198.51.100.12"
	)

	// AC5+AC1 boundary: 4 failures → 401; 5th failure → still 401; 6th attempt
	// (even with the CORRECT password) → 429 + Retry-After.
	for i := 0; i < loginMaxFailures-1; i++ {
		if rec := postLogin(h, ipA, "definitely-wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong %d: code=%d want 401 body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
	if rec := postLogin(h, ipA, "definitely-wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("5th failure must stay 401, got %d (AC5)", rec.Code)
	}
	rec := postLogin(h, ipA, loginPassword)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("6th attempt must be 429, got %d (AC1/AC2)", rec.Code)
	}
	if ra := rec.Header().Get("Retry-After"); ra == "" {
		t.Fatal("429 must carry Retry-After header (AC7)")
	}
	var payload struct {
		Error      string `json:"error"`
		RetryAfter int    `json:"retry_after"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.RetryAfter <= 0 {
		t.Fatalf("429 body must carry retry_after>0: body=%s err=%v", rec.Body.String(), err)
	}

	// AC6: malformed JSON → 400 and does NOT consume a failure slot. Fresh IP,
	// 4 wrongs, then a malformed request, then the 5th wrong must still be 401
	// (not blocked), proving the 400 was not counted.
	badReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("{not-json")))
	badReq.RemoteAddr = ipC + ":5555"
	badRec := httptest.NewRecorder()
	h.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("malformed body must be 400, got %d", badRec.Code)
	}
	for i := 0; i < loginMaxFailures-1; i++ {
		if r := postLogin(h, ipC, "definitely-wrong"); r.Code != http.StatusUnauthorized {
			t.Fatalf("ipC wrong %d: code=%d want 401", i+1, r.Code)
		}
	}
	if r := postLogin(h, ipC, "definitely-wrong"); r.Code != http.StatusUnauthorized {
		t.Fatalf("ipC 5th failure must be 401 (400s don't count, AC6), got %d", r.Code)
	}

	// AC4: success resets the counter. ipB does 3 wrongs, logs in successfully,
	// then 4 more wrongs must ALL be 401 — if the success hadn't reset, the
	// 2nd of those would already be a block (3+2=5).
	for i := 0; i < 3; i++ {
		if r := postLogin(h, ipB, "definitely-wrong"); r.Code != http.StatusUnauthorized {
			t.Fatalf("ipB wrong %d: code=%d want 401", i+1, r.Code)
		}
	}
	okRec := postLogin(h, ipB, loginPassword)
	if okRec.Code != http.StatusOK {
		t.Fatalf("correct password below threshold must be 200, got %d body=%s", okRec.Code, okRec.Body.String())
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(okRec.Body.Bytes(), &tok); err != nil || tok.Token == "" {
		t.Fatalf("login success must return token: body=%s", okRec.Body.String())
	}
	for i := 0; i < loginMaxFailures-1; i++ {
		if r := postLogin(h, ipB, "definitely-wrong"); r.Code != http.StatusUnauthorized {
			t.Fatalf("ipB post-reset wrong %d must be 401 (AC4), got %d", i+1, r.Code)
		}
	}

	// AC8: independent counters — ipB reached the threshold above, ipA was
	// already blocked, a brand-new IP must log in fine.
	ipD := "198.51.100.13"
	if r := postLogin(h, ipD, loginPassword); r.Code != http.StatusOK {
		t.Fatalf("fresh IP must be unaffected by others (AC8), got %d", r.Code)
	}
	_ = s // keep server alive until test end
}
