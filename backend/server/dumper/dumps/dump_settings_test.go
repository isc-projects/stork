package dumps_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dumper/dumps"
)

// Test that the dump is executed properly.
func TestSettingsDumpExecute(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db)
	_, _ = db.Model(&[]dbmodel.Setting{
		{
			Name:    "foo",
			ValType: dbmodel.SettingValTypeInt,
			Value:   "42",
		},
		{
			Name:    "password",
			ValType: dbmodel.SettingValTypePasswd,
			Value:   "secret",
		},
	}).Insert()
	dump := dumps.NewSettingsDump(db)

	// Act
	err := dump.Execute()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 1, dump.NumberOfArtifacts())

	artifact := dump.GetArtifact(0).(dumps.StructArtifact)
	artifactContent := artifact.GetStruct()
	settings, ok := artifactContent.(map[string]interface{})

	require.True(t, ok)
	require.EqualValues(t, 42, settings["foo"])
	_, ok = settings["password"]
	require.False(t, ok)
}
