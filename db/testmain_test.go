package db

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// testSuiteLockKey is the cross-package test-suite advisory lock (the same
// key the server and evidence TestMains hold, and which companies.go reserves
// from the company-lock keyspace).
const testSuiteLockKey int64 = 7337741002

// The db package tests share one postgres database with the agent package
// that runs before them (CI: go test -race -p 1 ./...). Truncate once before
// this package's tests start so companies/assets left by agent tests — or by
// any previous go test invocation — cannot skew attribution assertions like
// TestCompanyScopeAttribution.
//
// This file cannot use internal/testdb (testdb imports db, which would be an
// import cycle in test), so the lock+truncate dance is repeated inline,
// mirroring testdb.ResetAll: take the suite lock on a pinned connection and
// never release it — the process exit after m.Run() drops the session.
// Without a configured DSN the reset is skipped and the suite behaves exactly
// as before.
func TestMain(m *testing.M) {
	if dsn, _, err := DSN(); err == nil && dsn != "" {
		if pg, err := Open(dsn); err == nil {
			resetForTests(pg)
		}
	}
	os.Exit(m.Run())
}

// resetForTests pins one connection, takes the suite lock and truncates every
// table. The connection is intentionally left open: closing it would release
// the lock before the package finished running.
func resetForTests(pg *DB) {
	conn, err := pg.Conn(context.Background())
	if err != nil {
		pg.Close()
		return // unreachable database — the tests skip on their own
	}
	if _, err := conn.ExecContext(context.Background(),
		`SELECT pg_advisory_lock($1)`, testSuiteLockKey); err != nil {
		conn.Close()
		pg.Close()
		return
	}
	var tables string
	if err := conn.QueryRowContext(context.Background(), `SELECT string_agg(format('%I', tablename), ', ')
		FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables); err != nil || tables == "" {
		return
	}
	if _, err := conn.ExecContext(context.Background(),
		`TRUNCATE `+tables+` RESTART IDENTITY CASCADE`); err != nil {
		fmt.Fprintf(os.Stderr, "db tests: reset shared database: %v\n", err)
	}
}
