package dnsop

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the IP address cache can be populated and that the
// IP addresses are correctly mapped to the machines.
func TestMachineIPAddressCache(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create 5 machines with 2 network interfaces each. The IP addresses of
	// the next machine's first interface overlap with the previous machine's
	// second interface.
	for i := 0; i < 10; i += 2 {
		m := &dbmodel.Machine{
			Address:   "localhost",
			AgentPort: 8080 + int64(i),
			MachineNetworkInterfaces: []dbmodel.MachineNetworkInterface{
				{
					Name:            "eth0",
					Flags:           uint32(net.FlagUp),
					HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
					IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
						{IPAddress: fmt.Sprintf("192.168.1.%d/24", i+1)},
						{IPAddress: fmt.Sprintf("192.168.1.%d/24", i+2)},
					},
				},
				{
					Name:            "eth1",
					Flags:           uint32(net.FlagUp),
					HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
					IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
						{IPAddress: fmt.Sprintf("192.168.1.%d/24", i+3)},
						{IPAddress: fmt.Sprintf("192.168.1.%d/24", i+4)},
					},
				},
				{
					Name:            "lo",
					Flags:           uint32(net.FlagLoopback),
					HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
					IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
						{IPAddress: "127.0.0.1/8"},
						{IPAddress: "::1/128"},
					},
				},
			},
		}
		err := dbmodel.AddMachine(db, m)
		require.NoError(t, err)

		err = dbmodel.UpsertMachineNetworkInterfaces(db, m.ID, m.MachineNetworkInterfaces...)
		require.NoError(t, err)
	}

	// Fetch the IP addresses and store them in the cache.
	cache := newMachineIPAddressCache(db)
	err := cache.populate()
	require.NoError(t, err)

	// First address does not overlap with any other machine's IP addresses.
	machines, loopback := cache.getMachines("192.168.1.1")
	require.False(t, loopback)
	require.Len(t, machines, 1)
	require.EqualValues(t, 1, machines[0].ID)

	// Second address overlaps with the first machine's second interface.
	machines, loopback = cache.getMachines("192.168.1.2")
	require.False(t, loopback)
	require.Len(t, machines, 1)
	require.EqualValues(t, 1, machines[0].ID)

	// Third address overlaps with the second machine's first interface.
	machines, loopback = cache.getMachines("192.168.1.3")
	require.False(t, loopback)
	require.Len(t, machines, 2)

	// Fourth address overlaps with the second machine's second interface.
	machines, loopback = cache.getMachines("192.168.1.4")
	require.False(t, loopback)
	require.Len(t, machines, 2)

	// Loopback address is not included in the cache.
	machines, loopback = cache.getMachines("127.0.0.1")
	require.True(t, loopback)
	require.Empty(t, machines)

	machines, loopback = cache.getMachines("::1")
	require.True(t, loopback)
	require.Empty(t, machines)
}

// Test that if the cache is empty, the getMachines method returns an empty list.
func TestMachineIPAddressEmptyCache(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// There are no machines in the database, so the cache remains empty.
	cache := newMachineIPAddressCache(db)
	err := cache.populate()
	require.NoError(t, err)

	// Make sure that no machine is returned from the empty cache.
	machines, loopback := cache.getMachines("192.168.1.1")
	require.False(t, loopback)
	require.Empty(t, machines)
}

// Test that if the machine IP address lacks in the cache, the getMachines function
// can query the database and update the cache.
func TestMachineIPAddressCacheGetMachinesPullAddress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Populate the cache wity no machines.
	cache := newMachineIPAddressCache(db)
	err := cache.populate()
	require.NoError(t, err)

	// Add the machine with a single IP address.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
		MachineNetworkInterfaces: []dbmodel.MachineNetworkInterface{
			{
				Name:            "eth0",
				Flags:           uint32(net.FlagUp),
				HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
				IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
					{IPAddress: "192.168.1.1/24"},
				},
			},
		},
	}
	err = dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	err = dbmodel.UpsertMachineNetworkInterfaces(db, machine.ID, machine.MachineNetworkInterfaces...)
	require.NoError(t, err)

	// Query the cache for the IP address. It is not in the cache but the machine
	// with this IP address is present. It should be added to the cache.
	machines, loopback := cache.getMachines("192.168.1.1")
	require.False(t, loopback)
	require.Len(t, machines, 1)
	require.EqualValues(t, machine.ID, machines[0].ID)

	// Querying for the IP address that is not present in the database should
	// return an empty list.
	machines, loopback = cache.getMachines("192.168.1.2")
	require.False(t, loopback)
	require.Empty(t, machines)
}

// Test that the cache can be repopulated.
func TestMachineIPAddressCacheRepopulate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add the first machine with a single IP address.
	machine1 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
		MachineNetworkInterfaces: []dbmodel.MachineNetworkInterface{
			{
				Name:            "eth0",
				Flags:           uint32(net.FlagUp),
				HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
				IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
					{IPAddress: "192.168.1.1/24"},
				},
			},
		},
	}
	err := dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)
	err = dbmodel.UpsertMachineNetworkInterfaces(db, machine1.ID, machine1.MachineNetworkInterfaces...)
	require.NoError(t, err)

	// Populate the cache. It should now contain the first machine.
	cache := newMachineIPAddressCache(db)
	err = cache.populate()
	require.NoError(t, err)

	// Add the second machine with a single IP address.
	machine2 := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8081,
		MachineNetworkInterfaces: []dbmodel.MachineNetworkInterface{
			{
				Name:            "eth0",
				Flags:           uint32(net.FlagUp),
				HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
				IPAddresses: []dbmodel.MachineNetworkInterfaceIPAddress{
					{IPAddress: "192.168.1.2/24"},
				},
			},
		},
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)
	err = dbmodel.UpsertMachineNetworkInterfaces(db, machine2.ID, machine2.MachineNetworkInterfaces...)
	require.NoError(t, err)

	// Repopulate the cache. It should now contain both machines.
	err = cache.populate()
	require.NoError(t, err)

	// Make sure that both IP addresses and machines are present in the cache.
	machines, loopback := cache.getMachines("192.168.1.1")
	require.False(t, loopback)
	require.Len(t, machines, 1)
	require.EqualValues(t, machine1.ID, machines[0].ID)

	machines, loopback = cache.getMachines("192.168.1.2")
	require.False(t, loopback)
	require.Len(t, machines, 1)
	require.EqualValues(t, machine2.ID, machines[0].ID)
}
