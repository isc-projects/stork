package dbmigs

import (
	"log"
	"os"
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database"
)

// Expecting storktest database and the storktest user to have full privileges to it.
var testConnOptions = dbops.PgOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

var testDB *dbops.PgDB

// Setup a unit test with creating the schema.
func setupCreateSchema(t *testing.T) func (t *testing.T) {
	CreateSchema(t)
	return teardownTestCase;
}

// Setup a unit test without creating the schema.
func setupNoSchema(t *testing.T) func (t *testing.T) {
	TossSchema(t)
	return teardownTestCase;
}

// Tosses schema after the test.
func teardownTestCase(t *testing.T) {
	TossSchema(t)
}

// Create the database schema to the latest version.
func CreateSchema(t *testing.T) {
	TossSchema(t)
	_, _, err := Migrate(testDB, "init")
	require.NoError(t, err)
	_, _, err = Migrate(testDB, "up")
	require.NoError(t, err)
}

// Remove the database schema.
func TossSchema(t * testing.T) {
	_ = Toss(testDB)
}

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if addr, ok := os.LookupEnv("POSTGRES_ADDR"); ok {
		testConnOptions.Addr = addr
	}

	if testDB = dbops.NewPgDB(&testConnOptions); testDB == nil {
		log.Fatal("unable to create database instance")
	}
	defer testDB.Close()

	os.Exit(m.Run())
}

// Common function which tests a selected migration action.
func testMigrateAction(t *testing.T, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(testDB, action...)
	require.NoError(t, err)

	// Check that old database version has been returned as expected.
	require.Equal(t, expectedOldVersion, oldVersion)

	// Check that new database version has been returned as expected.
	require.Equal(t, expectedNewVersion, newVersion)
}

// Checks that schema version can be fetched from the database and
// that it is set to an expected value.
func testCurrentVersion(t *testing.T, expected int64) {
	current, err := CurrentVersion(testDB)
	require.NoError(t, err)
	require.Equal(t, expected, current)
}

// Test migrations between different database versions.
func TestMigrate(t *testing.T) {
	teardown := setupNoSchema(t)
	defer teardown(t)

	// Create versioning table in the database.
	testMigrateAction(t, 0, 0, "init")
	// Migrate from version 0 to version 1.
	testMigrateAction(t, 0, 1, "up", "1")
	// Migrate from version 1 to version 0.
	testMigrateAction(t, 1, 0, "down")
	// Migrate to version 1 again.
	testMigrateAction(t, 0, 1, "up", "1")
	// Check current version.
	testMigrateAction(t, 1, 1, "version")
	// Reset to the initial version.
	testMigrateAction(t, 1, 0, "reset")
}

// Test initialization and migration in a single step.
func TestInitMigrate(t *testing.T) {
	teardown := setupNoSchema(t)
	defer teardown(t)

	// Migrate from version 0 to version 1.
	testMigrateAction(t, 0, 1, "up", "1")
}

// Tests that the database schema can be initialized and migrated to the
// latest version with one call.
func TestInitMigrateToLatest(t *testing.T) {
	teardown := setupNoSchema(t)
	defer teardown(t)

	o, n, err := MigrateToLatest(testDB)
	require.NoError(t, err)
	require.Equal(t, int64(0), o)
	require.GreaterOrEqual(t, int64(2), n)
}

// Test that available schema version is returned as expected.
func TestAvailableVersion(t *testing.T) {
	teardown := setupCreateSchema(t)
	defer teardown(t)

	avail := AvailableVersion()
	require.GreaterOrEqual(t, avail, int64(2))
}

// Test that current version is returned from the database.
func TestCurrentVersion(t *testing.T) {
	teardown := setupNoSchema(t)
	defer teardown(t)

	// Initialize migrations.
	testMigrateAction(t, 0, 0, "init")
	// Initally, the version should be set to 0.
	testCurrentVersion(t, 0)
	// Go one version up.
	testMigrateAction(t, 0, 1, "up", "1")
	// Check that the current version is now set to 1.
	testCurrentVersion(t, 1)
}
