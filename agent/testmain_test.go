package agent

import (
	"fmt"
	"os"
	"testing"

	"github.com/Autumn-27/artex/internal/testdb"
)

// The agent package tests share one postgres database (db.DSN()) with the db
// and server test packages (CI runs everything with -p 1 in one go test
// invocation). testdb.ResetAll takes the cross-package suite advisory lock and
// truncates once before the package's tests start, so leftovers from other
// packages — or from a previous go test invocation (companies, assets,
// tasks…) — cannot skew assertions that count or scope rows. Within the
// package, tests run serially and clean up after themselves, matching CI
// where this package runs against a fresh database.
func TestMain(m *testing.M) {
	if err := testdb.ResetAll(); err != nil {
		fmt.Fprintf(os.Stderr, "agent tests: reset shared database: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
