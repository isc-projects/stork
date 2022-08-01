package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the global exclusions of the config checkers are returned properly.
func TestGetGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Act
	exclusions, err := GetGloballyExcludedCheckers(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, exclusions, 2)
	require.EqualValues(t, 1, exclusions[0].ID)
	require.EqualValues(t, "foo", exclusions[0].CheckerName)
	require.EqualValues(t, 2, exclusions[1].ID)
	require.EqualValues(t, "bar", exclusions[1].CheckerName)
}

// Test that an empty list is returned for missing the global exclusions of
// the config checkers.
func TestGetGloballyExcludedCheckersForEmptyData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	exclusions, err := GetGloballyExcludedCheckers(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, exclusions, 0)
}

// Test that the global exclusions of the config checkers are inserted properly.
func TestAddGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 2)
}

// Test that adding empty list of the global exclusions of the config checkers
// generates no error.
func TestAddGloballyExcludedCheckersForEmptyList(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 0)
}

// Test that adding the duplicated global exclusions of the config checkers
// generates an error.
func TestAddDuplicatedGloballyExcludedCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "foo"},
	})

	// Assert
	require.Error(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 0)
}

// Test that adding the same global exclusions of the config checkers
// in separate queries generates an error on the second call.
func TestAddDoubleGloballyExcludedCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err1 := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})
	err2 := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})

	// Assert
	require.NoError(t, err1)
	require.Error(t, err2)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 1)
}

// Test that the global exclusions of the config checkers are deleted properly.
func TestRemoveGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "bar"},
	})
	exclusions, _ := GetGloballyExcludedCheckers(db)

	// Act
	err := RemoveGloballyExcludedChekers(db, []*ConfigCheckerGlobalExclude{
		exclusions[1],
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ = GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 1)
	require.EqualValues(t, "foo", exclusions[0].CheckerName)
}
