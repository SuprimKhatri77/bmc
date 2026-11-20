package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	// blank import (_): we never call lib/pq ourselves, we only need its init() to run,
	// which registers the "postgres" driver with database/sql so sql.Open can find it
	// without the _ go fails with "imported and not used" (and the formatter removes the line)
	_ "github.com/lib/pq"
)

// hardcoded for now, later these will come from a config file / env vars
const (
	// driver name, has to match the name lib/pq registers itself under
	dbDriver = "postgres"
	// connection string for the local docker postgres (port 5433 on this machine, see the Makefile)
	dbSource = "postgresql://root:secret@localhost:5433/bmc?sslmode=disable"
)

// package-level so every test in package db can use them; TestMain fills them in before any test runs
// testQueries: the generated queries, for tests of single queries (accounts, entries, transfers)
var testQueries *Queries

// testDB: the connection pool itself (a real db, not a mock), for tests that need more than Queries,
// e.g. store_test.go builds a Store on it so it can start transactions
var testDB *sql.DB

// TestMain is the entry point for all the tests in this package: if it exists, go test calls it
// instead of running the tests directly, so setup (like opening the db) happens once here
// m (*testing.M) holds all the Test... functions of the package, m.Run() runs them and returns
// an exit code (0 = all passed), which os.Exit hands back to go test
func TestMain(m *testing.M) {
	// declared here so the line below can use = instead of :=
	// testDB, err := ... would create a new local testDB that hides the package-level one,
	// leaving the package-level testDB nil, so store_test.go would get a nil pool and crash
	var err error

	// sql.Open doesn't actually connect yet, it only checks the driver name and sets up a
	// connection pool; the real connection happens on the first query (testDB.Ping() forces one)
	testDB, err = sql.Open(dbDriver, dbSource)

	// so this only fails for an unknown driver or a bad dbSource, not when the db is down
	if err != nil {
		log.Fatal("cannot connect to db:", err)

	}

	// Ping makes a real connection, so the tests stop here with a clear error if the db is down
	// (instead of every test failing on its first query)
	if err := testDB.Ping(); err != nil {
		log.Fatal("cannot reach to db:", err)
	}

	// wrap the connection pool in Queries so the tests can call the generated methods on it
	testQueries = New(testDB)

	// run all the tests and exit with their result code
	os.Exit(m.Run())

}
