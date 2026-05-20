package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test inserting and updating IP addresses detected on the machine.
func TestUpsertMachineIPAddresses(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// Insert two IP addresses.
	err = UpsertMachineIPAddresses(db, m.ID, "192.168.1.1", "192.168.1.2")
	require.NoError(t, err)

	// Verify that the IP addresses are inserted.
	ipAddresses, err := GetMachineIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, ipAddresses, 2)
	require.Equal(t, "192.168.1.1", ipAddresses[0].IPAddress)
	require.Equal(t, "192.168.1.2", ipAddresses[1].IPAddress)

	// Preserve one of the IP addresses and replace the other one.
	err = UpsertMachineIPAddresses(db, m.ID, "192.168.1.1", "192.168.1.3")
	require.NoError(t, err)

	// Verify that the IP addresses are preserved and the other one is replaced.
	ipAddresses2, err := GetMachineIPAddresses(db, MachineIPAddressRelationMachine)
	require.NoError(t, err)
	require.Len(t, ipAddresses2, 2)
	require.Equal(t, "192.168.1.1", ipAddresses2[0].IPAddress)
	require.NotNil(t, ipAddresses2[0].Machine)
	require.Equal(t, "192.168.1.3", ipAddresses2[1].IPAddress)
	require.NotNil(t, ipAddresses2[1].Machine)

	// The ID of the first one should not change.
	require.Equal(t, ipAddresses[0].ID, ipAddresses2[0].ID)
	// The ID of the second one should change because it is replaced.
	require.NotEqual(t, ipAddresses[1].ID, ipAddresses2[1].ID)
}

// Test that no IP addresses are inserted when the list of IP addresses is empty.
func TestUpsertMachineIPAddressesEmpty(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// Insert an empty list of IP addresses.
	err = UpsertMachineIPAddresses(db, m.ID)
	require.NoError(t, err)

	// No addresses should be present in the database.
	ipAddresses, err := GetMachineIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, ipAddresses, 0)
}

// Test that inserting IP addresses followed by inserting an empty list
// of IP addresses removes machine's IP addresses from the database.
func TestUpsertMachineIPAddressesRemove(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)

	// Insert two IP addresses.
	err = UpsertMachineIPAddresses(db, m.ID, "192.168.1.1", "192.168.1.2")
	require.NoError(t, err)

	// Make sure that the IP addresses are present in the database.
	ipAddresses, err := GetMachineIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, ipAddresses, 2)
	require.Equal(t, "192.168.1.1", ipAddresses[0].IPAddress)
	require.Equal(t, "192.168.1.2", ipAddresses[1].IPAddress)

	// Insert an empty list of IP addresses.
	err = UpsertMachineIPAddresses(db, m.ID)
	require.NoError(t, err)

	// Make sure that the IP addresses are removed from the database.
	ipAddresses2, err := GetMachineIPAddresses(db)
	require.NoError(t, err)
	require.Empty(t, ipAddresses2)
}
