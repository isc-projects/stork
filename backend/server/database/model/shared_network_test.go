package dbmodel

import (
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"

	"testing"
)

// Tests that the shared network can be added and retrieved.
func TestAddSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
	require.NotZero(t, returned.Created)
}

// Tests that the shared network information can be updated.
func TestUpdateSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	network.Name = "different name"
	err = UpdateSharedNetwork(db, &network)
	require.NoError(t, err)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, network.Name, returned.Name)
}

// Tests that the shared network can be deleted.
func TestDeleteSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	network := SharedNetwork{
		Name: "funny name",
	}
	err := AddSharedNetwork(db, &network)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	err = DeleteSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.NotZero(t, network.ID)

	returned, err := GetSharedNetwork(db, network.ID)
	require.NoError(t, err)
	require.Nil(t, returned)
}
