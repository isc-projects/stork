package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the access point can be added to the database.
func TestAddAccessPoint(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	daemon := NewDaemon(m, "kea-dhcp4", true, nil)
	_ = AddDaemon(db, daemon)

	// Act
	err := addOrUpdateAccessPoint(db, &AccessPoint{
		DaemonID: daemon.ID,
		Type:     AccessPointControl,
		Address:  "foo",
		Port:     8000,
		Key:      "bar",
	})

	// Assert
	require.NoError(t, err)
	daemon, _ = GetDaemonByID(db, daemon.ID)
	require.Len(t, daemon.AccessPoints, 1)
	require.Equal(t, AccessPointControl, daemon.AccessPoints[0].Type)
	require.Equal(t, "foo", daemon.AccessPoints[0].Address)
	require.Equal(t, int64(8000), daemon.AccessPoints[0].Port)
	require.Equal(t, "bar", daemon.AccessPoints[0].Key)
}

// Test that the access point can be updated in the database.
func TestUpdateAccessPoint(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	daemon := NewDaemon(m, "kea-dhcp4", true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	_ = AddDaemon(db, daemon)

	// Act
	err := addOrUpdateAccessPoint(db, &AccessPoint{
		DaemonID: daemon.ID,
		Type:     AccessPointControl,
		Address:  "updated-address",
		Port:     9000,
		Key:      "updated-key",
	})

	// Assert
	require.NoError(t, err)
	daemon, _ = GetDaemonByID(db, daemon.ID)
	require.Len(t, daemon.AccessPoints, 1)
	require.Equal(t, "updated-address", daemon.AccessPoints[0].Address)
	require.Equal(t, int64(9000), daemon.AccessPoints[0].Port)
	require.Equal(t, "updated-key", daemon.AccessPoints[0].Key)
}

// Test that adding or updating an access point returns an error when the database
// operation fails.
func TestAddOrUpdateAccessPointDatabaseError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	daemon := NewDaemon(m, "kea-dhcp4", true, nil)
	_ = AddDaemon(db, daemon)

	// Act
	teardown()
	err := addOrUpdateAccessPoint(db, &AccessPoint{
		DaemonID: daemon.ID,
		Type:     AccessPointControl,
		Address:  "foo",
		Port:     8000,
		Key:      "bar",
	})

	// Assert
	require.Error(t, err)
}

// Test that deleting access points works correctly.
func TestDeleteAccessPoints(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	daemon := NewDaemon(m, "kea-dhcp4", true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "addr1",
			Port:    8000,
			Key:     "key1",
		},
		{
			Type:    AccessPointStatistics,
			Address: "addr2",
			Port:    9000,
			Key:     "key2",
		},
	})
	_ = AddDaemon(db, daemon)

	// Act
	err := deleteAccessPointsExcept(db, daemon.ID, []AccessPointType{AccessPointControl})
	require.NoError(t, err)

	// Assert
	daemon, _ = GetDaemonByID(db, daemon.ID)
	require.Len(t, daemon.AccessPoints, 1)
	require.Equal(t, AccessPointControl, daemon.AccessPoints[0].Type)
}
