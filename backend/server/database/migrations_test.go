package storkdb

import (
	"os"
	"path"
	"runtime"
	"testing"
)

// Expecting storktest database and the storktest user to have full privileges to it.
var testConnOptions = DbConnOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

// The migrations path defaults to current directory.
var migrationsPath string = "."

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Get the absolute path to the binary.
	_, filename, _, _ := runtime.Caller(0)

	// The schema files are in the schema subdirectory.
	migrationsPath = path.Join(path.Dir(filename), "schema/")

	// Toss the schema, including removal of the versioning table.
	Toss(&testConnOptions, migrationsPath)

	// Run tests.
	c := m.Run()
	os.Exit(c)
}

// Common function which tests a selected migration action.
func testMigrateAction(t *testing.T, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(&testConnOptions, migrationsPath, action...)
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
