package dbops

import (
	"log"
	"os"
	"testing"
	"github.com/stretchr/testify/require"
)

// Expecting storktest database and the storktest user to have full privileges to it.
var testConnOptions = PgOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

// Common function which cleans the environment before the tests.
func connDb() *PgDB {
	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if addr, ok := os.LookupEnv("POSTGRES_ADDR"); ok {
		testConnOptions.Addr = addr
	}

	db, err := NewPgDbConn(&testConnOptions);
	if  db == nil || err != nil {
		log.Fatalf("unable to create database instance %+v", err)
	}
	_ = Toss(db)

	return db
}

// Setup a unit test with creating the schema.
func setupCreateSchema(t *testing.T) *PgDB {
	db := connDb()
	_, _, err := Migrate(db, "init")
	require.NoError(t, err)
	_, _, err = Migrate(db, "up")
	require.NoError(t, err)
	return db
}

// Remove the database schema.
func cleanupDb(t *testing.T, db *PgDB) {
	err := Toss(db)
	require.NoError(t, err)
	db.Close()
}

// Common function which tests a selected migration action.
func testMigrateAction(t *testing.T, db *PgDB, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(db, action...)
	require.NoError(t, err)

	// Check that old database version has been returned as expected.
	require.Equal(t, expectedOldVersion, oldVersion)

	// Check that new database version has been returned as expected.
	require.Equal(t, expectedNewVersion, newVersion)
}

// Checks that schema version can be fetched from the database and
// that it is set to an expected value.
func testCurrentVersion(t *testing.T, db *PgDB, expected int64) {
	current, err := CurrentVersion(db)
	require.NoError(t, err)
	require.Equal(t, expected, current)
}

// Test migrations between different database versions.
func TestMigrate(t *testing.T) {
	db := connDb()
	defer cleanupDb(t, db)

	// Create versioning table in the database.
	testMigrateAction(t, db, 0, 0, "init")
	// Migrate from version 0 to version 1.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Migrate from version 1 to version 0.
	testMigrateAction(t, db, 1, 0, "down")
	// Migrate to version 1 again.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Check current version.
	testMigrateAction(t, db, 1, 1, "version")
	// Reset to the initial version.
	testMigrateAction(t, db, 1, 0, "reset")
}

// Test initialization and migration in a single step.
func TestInitMigrate(t *testing.T) {
	db := connDb()
	defer cleanupDb(t, db)

	// Migrate from version 0 to version 1.
	testMigrateAction(t, db, 0, 1, "up", "1")
}

// Tests that the database schema can be initialized and migrated to the
// latest version with one call.
func TestInitMigrateToLatest(t *testing.T) {
	db := connDb()
	defer cleanupDb(t, db)

	o, n, err := MigrateToLatest(db)
	require.NoError(t, err)
	require.Equal(t, int64(0), o)
	require.GreaterOrEqual(t, int64(4), n)
}

// Test that available schema version is returned as expected.
func TestAvailableVersion(t *testing.T) {
	db := setupCreateSchema(t)
	defer cleanupDb(t, db)

	avail := AvailableVersion()
	require.GreaterOrEqual(t, avail, int64(2))
}

// Test that current version is returned from the database.
func TestCurrentVersion(t *testing.T) {
	db := connDb()
	defer cleanupDb(t, db)

	// Initialize migrations.
	testMigrateAction(t, db, 0, 0, "init")
	// Initally, the version should be set to 0.
	testCurrentVersion(t, db, 0)
	// Go one version up.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Check that the current version is now set to 1.
	testCurrentVersion(t, db, 1)
}
