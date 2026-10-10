// Package testdb holds test-only helpers for packages whose tests run against
// the shared postgres database configured via db.DSN() (ARTEX_PG_DSN). It is
// imported exclusively from _test files / TestMain and never from production
// code.
package testdb

import (
	"context"
	"fmt"

	"github.com/Autumn-27/artex/db"
)

// suiteLockKey is the cross-package test-suite advisory lock already held by
// the server and evidence TestMains. Taking it here keeps concurrent
// `go test ./...` package binaries from truncating under each other; under
// CI's -p 1 it is uncontended.
const suiteLockKey int64 = 7337741002

// ResetAll takes the cross-package suite advisory lock and truncates every
// table in the configured database, so the calling test package starts from
// an empty schema. Rows left by earlier tests — in the same go test run or
// left over from a previous invocation — would otherwise be revived by
// server.New()'s restoreTaskRuntimes or skew row-count and attribution
// assertions. The next db.Open() re-seeds builtin rows (agents, prompt
// variables), so truncation is safe.
//
// The advisory lock is taken on a pinned connection and intentionally never
// released: the caller's TestMain os.Exit()s right after m.Run(), and the
// session drop releases it (the same convention as the server/evidence
// TestMains, whose deferred unlocks are likewise skipped by os.Exit). Closing
// the connection early would hand the lock to the next package mid-run.
//
// Without a configured DSN — or when the database is unreachable — ResetAll
// is a no-op: the caller's tests skip on their own in that case.
func ResetAll() error {
	dsn, _, err := db.DSN()
	if err != nil {
		return nil // no DSN configured — tests will skip on their own
	}
	pg, err := db.Open(dsn)
	if err != nil {
		return nil // same: an unreachable database only means tests will skip
	}
	conn, err := pg.Conn(context.Background())
	if err != nil {
		pg.Close()
		return nil
	}
	if _, err := conn.ExecContext(context.Background(),
		`SELECT pg_advisory_lock($1)`, suiteLockKey); err != nil {
		conn.Close()
		pg.Close()
		return fmt.Errorf("acquire suite lock: %w", err)
	}
	var tables string
	if err := conn.QueryRowContext(context.Background(), `SELECT string_agg(format('%I', tablename), ', ')
		FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables); err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	if tables == "" {
		return nil
	}
	if _, err := conn.ExecContext(context.Background(),
		`TRUNCATE `+tables+` RESTART IDENTITY CASCADE`); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	return nil
}
