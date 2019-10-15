package storkdb

import (
	"os"
	"path"
	"runtime"
	"testing"
)

var testConnOptions = DbConnOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

var migrationsPath string = "."

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	migrationsPath = path.Join(path.Dir(filename), "schema/")

	Toss(&testConnOptions, migrationsPath)
	c := m.Run()
	os.Exit(c)
}

func testMigrateAction(t *testing.T, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(&testConnOptions, migrationsPath, action...)
	if err != nil {
		t.Fatalf("migration failed with error %s", err.Error())
	}

	if oldVersion != expectedOldVersion {
		t.Errorf("expected old version %d, got %d", expectedOldVersion, oldVersion)
	}

	if newVersion != expectedNewVersion {
		t.Errorf("expected new version %d, got %d", expectedNewVersion, newVersion)
	}
}

func TestMigrate(t *testing.T) {
	testMigrateAction(t, 0, 0, "init")
	testMigrateAction(t, 0, 1, "up", "1")
	testMigrateAction(t, 1, 0, "down")
	testMigrateAction(t, 0, 1, "up", "1")
	testMigrateAction(t, 1, 1, "version")
	testMigrateAction(t, 1, 0, "reset")
}
