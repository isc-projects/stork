package dump_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dumppkg "isc.org/stork/server/dumper/dump"
)

// Test that the dump is executed properly.
func TestSettingsDumpExecute(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)
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
	dump := dumppkg.NewSettingsDump(db)

	// Act
	err := dump.Execute()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 1, dump.GetArtifactsNumber())

	artifact := dump.GetArtifact(0).(dumppkg.StructArtifact)
	artifactContent := artifact.GetStruct()
	settings, ok := artifactContent.(map[string]interface{})

	require.True(t, ok)
	require.EqualValues(t, 42, settings["foo"])
	_, ok = settings["password"]
	require.False(t, ok)
}
