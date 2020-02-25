package dbops

import (
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"
)

// Creates new instance of the database connection.
func createNewPgDB() (*pg.DB, error) {
	addr := os.Getenv("POSTGRES_ADDR")

	var host, port string
	if addr != "" {
		host, port, _ = net.SplitHostPort(addr)
	} else {
		host = "localhost"
		port = "5432"
	}
	portInt, _ := strconv.Atoi(port)

	settings := DatabaseSettings{
		BaseDatabaseSettings: BaseDatabaseSettings{
			DbName: "storktest",
			User:   "storktest",
			Host:   host,
			Port:   portInt,
		},
	}
	os.Setenv("STORK_DATABASE_PASSWORD", "storktest")

	db, err := NewPgDB(&settings)
	return db, err
}

// Test that new database connection instance is successfully created.
func TestNewPgDB(t *testing.T) {
	db, err := createNewPgDB()
	defer os.Setenv("STORK_DATABASE_PASSWORD", "")

	require.NotNil(t, db)
	require.NoError(t, err)
}

// Tests the logic that creates new transaction or returns an
// existing one.
func TestTransaction(t *testing.T) {
	db, err := createNewPgDB()
	defer os.Setenv("STORK_DATABASE_PASSWORD", "")
	require.NotNil(t, db)
	require.NoError(t, err)

	// Start new transaction.
	tx, rollback, commit, err := Transaction(db)
	require.NotNil(t, tx)
	require.NoError(t, err)
	require.NotNil(t, rollback)
	require.NotNil(t, commit)
	// Check that the commit operation returns no error.
	err = commit()
	require.NoError(t, err)

	// Start new transaction here.
	tx, err = db.Begin()
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback()
	}()
	require.NotNil(t, tx)

	// This time pass the transaction to the function under test. The function
	// should determine that the transaction was already started and return
	// it back to the caller.
	tx2, rollback, commit, err := Transaction(tx)
	require.NoError(t, err)
	require.NotNil(t, rollback)
	defer rollback()
	require.NotNil(t, tx2)
	require.NotNil(t, commit)
	// Those two pointers should point at the same object.
	require.Same(t, tx, tx2)
}
