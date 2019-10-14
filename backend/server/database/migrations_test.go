package storkdb

import (
	"os"
	"testing"
)

var testConnOptions = DbConnOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

func TestMain(m *testing.M) {
	Toss(&testConnOptions)
	c := m.Run()
	os.Exit(c)
}

func testMigrateAction(t *testing.T, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := Migrate(&testConnOptions, action...)
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
