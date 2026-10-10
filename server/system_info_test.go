package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSystemInfoEndpoint exercises GET /api/system/info through the real HTTP
// mux (requireAuth middleware included) and asserts the acceptance criteria
// from spec/features/system-info-endpoint.md:
//
//	AC1 200 + 6 fields when authenticated
//	AC2 401 when unauthenticated
//	AC3 version == server.BuildVersion
//	AC4 uptime monotonically non-decreasing across two calls
//	AC5 response contains no blacklist fields (env/secret/path/hostname/api_key)
//	AC6 /api/health response shape unchanged (regression)
func TestSystemInfoEndpoint(t *testing.T) {
	// Save and restore BuildVersion to keep the global package state clean for
	// parallel test packages. AC3 specifically checks that the field reflects
	// the current BuildVersion value, so we set a recognizable sentinel.
	orig := BuildVersion
	defer func() { BuildVersion = orig }()
	BuildVersion = "vtest-9.9.9"

	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	td := t.TempDir()
	s := New(context.Background(), m, td, td, td)
	h := s.Handler()
	tok, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	// allowedFields is the closed-set whitelist from plan-spec §4. Any other
	// top-level key is a leak and fails AC5.
	allowedFields := map[string]struct{}{
		"version":    {},
		"go_version": {},
		"go_os":      {},
		"go_arch":    {},
		"start_time": {},
		"uptime":     {},
	}
	blacklistSubstrs := []string{"env", "secret", "path", "hostname", "api_key", "apikey"}

	// authedGet issues GET with a Bearer token through the full mux.
	authedGet := func(path string) (*httptest.ResponseRecorder, map[string]any) {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}

	// --- AC2: unauthenticated → 401 ---
	reqNoAuth := httptest.NewRequest("GET", "/api/system/info", nil)
	recNoAuth := httptest.NewRecorder()
	h.ServeHTTP(recNoAuth, reqNoAuth)
	if recNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("AC2: unauthenticated want 401, got %d (body=%s)", recNoAuth.Code, recNoAuth.Body.String())
	}

	// --- AC1: authenticated → 200 + 6 fields ---
	rec1, out1 := authedGet("/api/system/info")
	if rec1.Code != 200 {
		t.Fatalf("AC1: want 200, got %d (body=%s)", rec1.Code, rec1.Body.String())
	}
	if len(out1) != len(allowedFields) {
		t.Fatalf("AC1: want exactly %d fields, got %d (%v)", len(allowedFields), len(out1), keysOf(out1))
	}
	for k := range out1 {
		if _, ok := allowedFields[k]; !ok {
			t.Fatalf("AC5/AC1: unexpected field %q (allowed set = %v)", k, keysOf(allowedFields))
		}
	}
	for _, f := range []string{"version", "go_version", "go_os", "go_arch", "start_time", "uptime"} {
		if _, ok := out1[f]; !ok {
			t.Fatalf("AC1: missing field %q in response %v", f, out1)
		}
	}

	// --- AC3: version == BuildVersion ---
	if v, _ := out1["version"].(string); v != BuildVersion {
		t.Fatalf("AC3: version=%q want %q", v, BuildVersion)
	}

	// --- AC4: uptime monotonically non-decreasing ---
	up1, _ := out1["uptime"].(float64)
	// Sleep just enough that time.Since advances beyond float64 noise; keep
	// well under the AC4 < 1s tolerance band in the feature-spec.
	time.Sleep(50 * time.Millisecond)
	_, out2 := authedGet("/api/system/info")
	up2, _ := out2["uptime"].(float64)
	if up2 < up1 {
		t.Fatalf("AC4: uptime must be non-decreasing: up1=%v up2=%v", up1, up2)
	}
	if up1 < 0 || up2 < 0 {
		t.Fatalf("AC4: uptime must be non-negative: up1=%v up2=%v", up1, up2)
	}

	// --- AC5: response body contains no blacklisted top-level fields ---
	// The closed-set check in AC1 already guarantees only the 6 allowed keys
	// appear; blacklistSubstrs are listed for documentation and future
	// substring-level regression tests. AC5 is satisfied by AC1's key-set
	// assertion: any blacklisted field name would have failed there.
	_ = blacklistSubstrs

	// --- AC6: /api/health regression — still {ok, service, version} ---
	recH, outH := authedGet("/api/health")
	if recH.Code != 200 {
		t.Fatalf("AC6: /api/health want 200, got %d", recH.Code)
	}
	wantH := map[string]struct{}{"ok": {}, "service": {}, "version": {}}
	if len(outH) != len(wantH) {
		t.Fatalf("AC6: /api/health field set drifted: got %v want %v", keysOf(outH), keysOf(wantH))
	}
	for k := range wantH {
		if _, ok := outH[k]; !ok {
			t.Fatalf("AC6: /api/health missing field %q", k)
		}
	}
}

// keysOf returns the keys of m as a slice, for readable test failure output.
// Generic over value type so it accepts both map[string]any (response body)
// and map[string]struct{} (whitelist sets).
func keysOf[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
