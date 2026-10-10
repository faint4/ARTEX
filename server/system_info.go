package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// startTime captures when the server package was initialized. Used by
// /api/system/info to report uptime. It is set once at package init time
// (earlier than any init() function), which is a close enough approximation
// of "process start" for diagnostic purposes — the feature-spec allows < 1s
// tolerance.
//
// Read-only after init; never mutated.
var startTime = time.Now()

// systemInfo handles GET /api/system/info and returns build + runtime
// diagnostics for the ARTEX backend.
//
// This endpoint is intentionally SEPARATE from /api/health:
//   - /api/health stays minimal ({ok, service, version}) for liveness probes
//     and is exempted from JWT auth (see auth.go requireAuth whitelist).
//   - /api/system/info returns richer diagnostic fields (go_version, go_os,
//     go_arch, start_time, uptime) and is NOT whitelisted, so it requires
//     a valid JWT — preventing unauthenticated information disclosure.
//
// FIELD WHITELIST (see plan-spec §4): only the 6 fields below may appear in
// the response. Do NOT add env vars, file paths, hostnames, or any secret
// material — the test in system_info_test.go enforces this closed set.
func (s *Server) systemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"version":    BuildVersion,
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"start_time": startTime.UTC().Format(time.RFC3339),
		"uptime":     time.Since(startTime).Seconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
