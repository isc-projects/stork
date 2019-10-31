package dbmigs

import (
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

func SetupDatabaseTestCase(t *testing.T) func (t *testing.T) {
	CreateSchema(t)
	return func (t *testing.T) {
		TossSchema(t)
	}
}

// Create the database schema to the latest version.
func CreateSchema(t *testing.T) {
	TossSchema(t)
	_, _, err := Migrate(&testConnOptions, "init")
	require.NoError(t, err)
	_, _, err = Migrate(&testConnOptions, "up")
	require.NoError(t, err)
}

// Remove the database schema.
func TossSchema(t * testing.T) {
	_ = Toss(&testConnOptions)
}

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if addr, ok := os.LookupEnv("POSTGRES_ADDR"); ok {
		testConnOptions.Addr = addr
	}
}

// Common function which tests a selected migration action.
func testMigrateAction(t *testing.T, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(&testConnOptions, action...)
	if err != nil {
		t.Fatalf("migration failed with error %s", err.Error())
	}

	// Check that old database version has been returned as expected.
	if oldVersion != expectedOldVersion {
		t.Errorf("expected old version %d, got %d", expectedOldVersion, oldVersion)
	}

	// Check that new database version has been returned as expected.
	if newVersion != expectedNewVersion {
		t.Errorf("expected new version %d, got %d", expectedNewVersion, newVersion)
	}
}

// Checks that schema version can be fetched from the database and
// that it is set to an expected value.
func testCurrentVersion(t *testing.T, expected int64) {
	current, err := CurrentVersion(&testConnOptions)
	if err != nil {
		t.Fatalf("getting current version failed with error %s", err.Error())
	}

	if current != expected {
		t.Errorf("expected current version %d, got %d", expected, current)
	}
}

// Test migrations between different database versions.
func TestMigrate(t *testing.T) {
	teardown := SetupDatabaseTestCase(t)
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

// Test that available schema version is returned as expected.
func TestAvailableVersion(t *testing.T) {
	teardown := SetupDatabaseTestCase(t)
	defer teardown(t)

	avail := AvailableVersion()

	var expected int64 = 2
	if avail != expected {
		t.Errorf("expected available version %d, got %d", expected, avail)
	}
}

// Test that current version is returned from the database.
func TestCurrentVersion(t *testing.T) {
	teardown := SetupDatabaseTestCase(t)
	defer teardown(t)

	// Initally, the version should be set to 0.
	testCurrentVersion(t, 0)
	// Go one version up.
	testMigrateAction(t, 0, 1, "up", "1")
	// Check that the current version is now set to 1.
	testCurrentVersion(t, 1)
}
