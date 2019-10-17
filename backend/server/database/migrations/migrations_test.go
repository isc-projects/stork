package dbmigs

import (
	"os"
	_ "path"
	_ "runtime"
	"testing"
)

// Expecting storktest database and the storktest user to have full privileges to it.
var testConnOptions = DbConnOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if _, ok := os.LookupEnv("POSTGRES_DB"); ok {
		testConnOptions.Addr = "postgres:5432"
	}

	// Toss the schema, including removal of the versioning table.
	Toss(&testConnOptions)

	// Run tests.
	c := m.Run()
	os.Exit(c)
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

// Test migrations between different database versions.
func TestMigrate(t *testing.T) {
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
