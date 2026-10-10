package server

// Shared helpers for tests that exercise a full Server against real postgres
// (db.DSN() / ARTEX_PG_DSN). All tests in this package run in ONE process
// (CI: go test -race -p 1) and share ONE database, so two invariants keep the
// suite race-free:
//
//  1. Every background goroutine started by New (scheduler, notifier, evidence
//     GC, login-rate GC, engine planner/worker loops) must STOP before the test
//     ends. New(context.Background()) leaks them into later tests, where they
//     keep mutating state and race with the agent package globals that the next
//     New() rewrites (agent.PromptOverride & friends) — the source of the
//     whack-a-mole race failures tracked in Issue #4.
//  2. Each test must start from a clean database. Rows left by earlier tests
//     (tasks, LLM profiles, conversations…) are revived by the next New()'s
//     restoreTaskRuntimes, which starts REAL agent runs mid-test against the
//     shared DB.
//
// newTestServer enforces both; use it instead of calling New directly.

import (
	"context"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
)

// resetSharedDB truncates every table in the shared test database, so the next
// Open sees no rows from previous tests — or from previous packages (db, agent
// and llmrec tests share the same DSN). Safe under -p 1: only one test runs at
// a time. Runs BEFORE the seeding NewManager, so db.Open re-seeds the builtin
// rows (agents, prompt vars) into the emptied tables.
func resetSharedDB(t *testing.T) {
	t.Helper()
	dsn, _, err := db.DSN()
	if err != nil {
		return // no DSN configured — the caller's NewManager would skip anyway
	}
	pg, err := db.Open(dsn)
	if err != nil {
		return // same: tests against an unreachable DB skip on their own
	}
	defer pg.Close()
	var tables string
	if err := pg.QueryRow(`SELECT string_agg(format('%I', tablename), ', ')
		FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables); err != nil || tables == "" {
		return
	}
	if _, err := pg.Exec(`TRUNCATE ` + tables + ` RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset shared test database: %v", err)
	}
}

// newTestServer builds a full Server + Manager backed by a freshly reset
// database, with cleanup guarantees:
//   - truncates the shared DB first (see resetSharedDB),
//   - binds every background goroutine to a per-test cancellable context,
//   - on cleanup: cancel → settle → close the Manager, in that order, and all
//     BEFORE t.TempDir removal (t.Cleanup runs LIFO after the defers).
//
// The 500ms settle covers goroutines already past their last select when cancel
// fires — engine loops use sleepCtx, so they exit at the next select.
func newTestServer(t *testing.T, dirs ...string) (*Server, *Manager) {
	t.Helper()
	resetSharedDB(t)
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	if m.pg == nil {
		t.Skip("postgres unavailable (no DSN) — skipping")
	}
	for len(dirs) < 3 {
		dirs = append(dirs, t.TempDir())
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := New(ctx, m, dirs[0], dirs[1], dirs[2])
	t.Cleanup(func() {
		cancel()
		time.Sleep(500 * time.Millisecond)
		m.Close()
	})
	return s, m
}
