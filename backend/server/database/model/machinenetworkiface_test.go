package dbmodel

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test inserting and updating network interfaces detected on the machine.
func TestUpsertMachineNetworkInterfaces(t *testing.T) {
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
	interfaces := []MachineNetworkInterface{
		{
			Name:            "eth0",
			Flags:           uint32(net.FlagUp),
			HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
			IPAddresses: []MachineNetworkInterfaceIPAddress{
				{IPAddress: "192.168.1.1"},
				{IPAddress: "192.168.1.2"},
			},
		},
		{
			Name:            "eth1",
			Flags:           uint32(net.FlagUp),
			HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
			IPAddresses:     []MachineNetworkInterfaceIPAddress{{IPAddress: "192.168.1.3"}, {IPAddress: "192.168.1.4"}},
		},
	}
	err = UpsertMachineNetworkInterfaces(db, m.ID, interfaces...)
	require.NoError(t, err)

	// Verify that the network interfaces are inserted.
	ipAddresses, err := GetMachineNetworkInterfaceIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, ipAddresses, 4)
	require.Equal(t, "192.168.1.1", ipAddresses[0].IPAddress)
	require.Equal(t, "192.168.1.2", ipAddresses[1].IPAddress)
	require.Equal(t, "192.168.1.3", ipAddresses[2].IPAddress)
	require.Equal(t, "192.168.1.4", ipAddresses[3].IPAddress)

	// Preserve one of the interfaces and replace the other one.
	interfaces[1] = MachineNetworkInterface{
		Name:            "eth2",
		Flags:           uint32(net.FlagUp),
		HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
		IPAddresses: []MachineNetworkInterfaceIPAddress{
			{IPAddress: "192.168.1.5"},
			{IPAddress: "192.168.1.6"},
		},
	}
	err = UpsertMachineNetworkInterfaces(db, m.ID, interfaces...)
	require.NoError(t, err)

	// Verify that the first interface is preserved and the second one is replaced.
	ipAddresses2, err := GetMachineNetworkInterfaceIPAddresses(db, MachineNetworkInterfaceIPAddressRelationInterface)
	require.NoError(t, err)
	require.Len(t, ipAddresses2, 4)
	require.Equal(t, "192.168.1.1", ipAddresses2[0].IPAddress)
	require.Equal(t, "192.168.1.2", ipAddresses2[1].IPAddress)
	require.Equal(t, "192.168.1.5", ipAddresses2[2].IPAddress)
	require.Equal(t, "192.168.1.6", ipAddresses2[3].IPAddress)

	// Make sure that IDs of the interface IDs for the first two IP addresses remain the same.
	require.Equal(t, ipAddresses[0].MachineNetworkInterfaceID, ipAddresses2[0].MachineNetworkInterfaceID)
	require.Equal(t, ipAddresses[1].MachineNetworkInterfaceID, ipAddresses2[1].MachineNetworkInterfaceID)
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
	err = UpsertMachineNetworkInterfaces(db, m.ID)
	require.NoError(t, err)

	// No addresses should be present in the database.
	interfaces, err := GetMachineNetworkInterfaceIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, interfaces, 0)
}

// Test that inserting network interfaces followed by inserting an empty list
// of interfaces removes machine's interfaces and the IP addresses from the
// database.
func TestUpsertMachineHostInterfacesRemove(t *testing.T) {
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
	err = UpsertMachineNetworkInterfaces(db, m.ID, MachineNetworkInterface{
		Name:            "eth0",
		Flags:           uint32(net.FlagUp),
		HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
		IPAddresses:     []MachineNetworkInterfaceIPAddress{{IPAddress: "192.168.1.1"}, {IPAddress: "192.168.1.2"}},
	})
	require.NoError(t, err)

	// Make sure that the IP addresses are present in the database.
	addrs, err := GetMachineNetworkInterfaceIPAddresses(db)
	require.NoError(t, err)
	require.Len(t, addrs, 2)
	require.Equal(t, "192.168.1.1", addrs[0].IPAddress)
	require.Equal(t, "192.168.1.2", addrs[1].IPAddress)

	// Insert an empty list of interfaces.
	err = UpsertMachineNetworkInterfaces(db, m.ID)
	require.NoError(t, err)

	// Make sure that the network interfaces are removed from the database.
	interfaces2, err := GetMachineNetworkInterfaceIPAddresses(db)
	require.NoError(t, err)
	require.Empty(t, interfaces2)
}

// Test that the machines including the specified IP address on one of their network interfaces
// are returned.
func TestGetMachinesByNetworkInterfaceIPAddress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create 5 machines with 2 network interfaces each. The IP addresses of
	// the next machine's first interface overlap with the previous machine's
	// second interface.
	for i := 0; i < 10; i += 2 {
		m := &Machine{
			Address:   "localhost",
			AgentPort: 8080 + int64(i),
			MachineNetworkInterfaces: []MachineNetworkInterface{
				{
					Name:            "eth0",
					Flags:           uint32(net.FlagUp),
					HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
					IPAddresses: []MachineNetworkInterfaceIPAddress{
						{IPAddress: fmt.Sprintf("192.168.1.%d", i+1)},
						{IPAddress: fmt.Sprintf("192.168.1.%d", i+2)},
					},
				},
				{
					Name:            "eth1",
					Flags:           uint32(net.FlagUp),
					HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
					IPAddresses: []MachineNetworkInterfaceIPAddress{
						{IPAddress: fmt.Sprintf("192.168.1.%d", i+3)},
						{IPAddress: fmt.Sprintf("192.168.1.%d", i+4)},
					},
				},
			},
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		err = UpsertMachineNetworkInterfaces(db, m.ID, m.MachineNetworkInterfaces...)
		require.NoError(t, err)
	}

	t.Run("overlapping IP addresses", func(t *testing.T) {
		// Get machine by IP address.
		machines, err := GetMachinesByNetworkInterfaceIPAddress(db, "192.168.1.3", MachineRelationNetworkInterfacesIPAddresses)
		require.NoError(t, err)

		// Because the same IP address is present on the two machines.
		require.Len(t, machines, 2)

		// The first machine contains the IP address on the eth1 interface.
		require.EqualValues(t, 1, machines[0].ID)
		// The second machine contains the IP address on the eth0 interface.
		require.EqualValues(t, 2, machines[1].ID)

		// Each machine has 2 network interfaces.
		require.Len(t, machines[0].MachineNetworkInterfaces, 2)

		// The first interface is eth0 according to the sort order.
		require.Equal(t, "eth0", machines[0].MachineNetworkInterfaces[0].Name)
		require.Len(t, machines[0].MachineNetworkInterfaces[0].IPAddresses, 2)
		require.Equal(t, "192.168.1.1", machines[0].MachineNetworkInterfaces[0].IPAddresses[0].IPAddress)
		require.Equal(t, "192.168.1.2", machines[0].MachineNetworkInterfaces[0].IPAddresses[1].IPAddress)
		// Next is the eth1 interface.
		require.Equal(t, "eth1", machines[0].MachineNetworkInterfaces[1].Name)
		require.Len(t, machines[0].MachineNetworkInterfaces[1].IPAddresses, 2)
		require.Equal(t, "192.168.1.3", machines[0].MachineNetworkInterfaces[1].IPAddresses[0].IPAddress)
		require.Equal(t, "192.168.1.4", machines[0].MachineNetworkInterfaces[1].IPAddresses[1].IPAddress)

		// Check the second machine.
		require.EqualValues(t, 2, machines[1].ID)
		require.Len(t, machines[1].MachineNetworkInterfaces, 2)
		require.Equal(t, "eth0", machines[1].MachineNetworkInterfaces[0].Name)
		require.Len(t, machines[1].MachineNetworkInterfaces[0].IPAddresses, 2)
		require.Equal(t, "192.168.1.3", machines[1].MachineNetworkInterfaces[0].IPAddresses[0].IPAddress)
		require.Equal(t, "192.168.1.4", machines[1].MachineNetworkInterfaces[0].IPAddresses[1].IPAddress)
		require.Equal(t, "eth1", machines[1].MachineNetworkInterfaces[1].Name)
		require.Len(t, machines[1].MachineNetworkInterfaces[1].IPAddresses, 2)
		require.Equal(t, "192.168.1.5", machines[1].MachineNetworkInterfaces[1].IPAddresses[0].IPAddress)
		require.Equal(t, "192.168.1.6", machines[1].MachineNetworkInterfaces[1].IPAddresses[1].IPAddress)
	})

	t.Run("non-overlapping IP addresses", func(t *testing.T) {
		// Get machine by IP address.
		machines, err := GetMachinesByNetworkInterfaceIPAddress(db, "192.168.1.1", MachineRelationNetworkInterfacesIPAddresses)
		require.NoError(t, err)

		// The IP address is present on the first machine.
		require.Len(t, machines, 1)
		require.EqualValues(t, 1, machines[0].ID)
		require.Len(t, machines[0].MachineNetworkInterfaces, 2)
	})

	t.Run("non-existing IP address", func(t *testing.T) {
		// Get machine by non-existingIP address.
		machines, err := GetMachinesByNetworkInterfaceIPAddress(db, "1.1.1.1", MachineRelationNetworkInterfacesIPAddresses)
		require.NoError(t, err)

		// The IP address is not present on any machine.
		require.Empty(t, machines)
	})

	t.Run("no relations", func(t *testing.T) {
		// Get machine by IP address.
		machines, err := GetMachinesByNetworkInterfaceIPAddress(db, "192.168.1.1")
		require.NoError(t, err)

		// The IP address is present on the first machine.
		require.Len(t, machines, 1)
		require.EqualValues(t, 1, machines[0].ID)
		require.Empty(t, machines[0].MachineNetworkInterfaces)
	})

	t.Run("no IP address relation", func(t *testing.T) {
		// Get machine by IP address.
		machines, err := GetMachinesByNetworkInterfaceIPAddress(db, "192.168.1.1", MachineRelationNetworkInterfaces)
		require.NoError(t, err)

		// The IP address is present on the first machine.
		require.Len(t, machines, 1)
		require.EqualValues(t, 1, machines[0].ID)
		require.Len(t, machines[0].MachineNetworkInterfaces, 2)
		require.Empty(t, machines[0].MachineNetworkInterfaces[0].IPAddresses)
		require.Empty(t, machines[0].MachineNetworkInterfaces[1].IPAddresses)
	})
}

// This benchmark tests performance of a query that returns machines holding the
// specified IP address on one of their network interfaces. The benchmark creates
// a variable number of machines, each with 4 network interfaces. The total number
// of IP addresses on the machine is 10. The test cases differ by the total number
// of IP addresses. The number of machines is 10 times lower. For example, for
// 10000 IP addresses, there are 1000 machines.
//
// This benchmark was used to decide on the database schema for holding network
// interfaces and IP addresses. There were three different approaches tested:
//
//   - No new tables, the network interfaces and IP addresses were stored in the
//     machine table as JSONB.
//
//   - A new table for network interfaces. The IP addresses were stored in the
//     table as INET[] array.
//
//   - Normalized schema with two tables: machine_network_interface and
//     machine_network_interface_ip_address (current solution).
//
// Out of these three approaches, the first one was discarded because it appeared
// very slow in comparison to the other two approaches.
//
// The second approach was considered due to its simplicity. However, it turned out
// that the performance degraded significantly when the number of interfaces was
// increasing. For example, doubling the number of interfaces increased the query
// time by 1.5x. For 10000 IP addresses (and 1000 machines), we obtained the
// following results for the second approach:
//
//	726198 ns/op      594139 B/op        131 allocs/op
//
// The third approach was chosen because it provides the best and most predictable
// performance, with the following benchmark results:
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/10_IP_addresses-12
//
//	341410 ns/op	  617438 B/op	     127 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/100_IP_addresses-12
//
//	356268 ns/op	  623173 B/op	     127 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/1000_IP_addresses-12
//
//	351932 ns/op	  617530 B/op	     127 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/10000_IP_addresses-12
//
//	376051 ns/op	  624253 B/op	     127 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/20000_IP_addresses-12
//
//	597063 ns/op	  585125 B/op	     126 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/40000_IP_addresses-12
//
//	392546 ns/op	  611410 B/op	     126 allocs/op
//
// BenchmarkGetMachinesByNetworkInterfaceIPAddress/65535_IP_addresses-12
//
//	370520 ns/op	  609787 B/op	     127 allocs/op
//
// The above results clearly show that the normalized schema has stable
// performance, and it is the best choice for hot path queries by IP address.
//
// Note that the benchmark always gets the same IP address. Randomizing the
// queries IP address would increase the results spread, but the conclusions
// would remain the same.
func BenchmarkGetMachinesByNetworkInterfaceIPAddress(b *testing.B) {
	// Each test case creates the specified number of IP addresses.
	tests := []int{10, 100, 1000, 10000, 20000, 40000, 65535}
	for _, test := range tests {
		b.Run(fmt.Sprintf("%d IP addresses", test), func(b *testing.B) {
			db, _, teardown := dbtest.SetupDatabaseTestCase(b)
			defer teardown()

			for i := 0; i < test; i += 10 {
				m := &Machine{
					ID:        0,
					Address:   "localhost",
					AgentPort: 8080 + int64(i),
					MachineNetworkInterfaces: []MachineNetworkInterface{
						{
							Name:            "eth0",
							Flags:           uint32(net.FlagUp),
							HardwareAddress: []byte{1, 2, 3, 4, 5, 6},
							IPAddresses: []MachineNetworkInterfaceIPAddress{
								{IPAddress: fmt.Sprintf("192.168.%d.%d", i&0xFF00>>8, i&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+1)&0xFF00>>8, (i+1)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+2)&0xFF00>>8, (i+2)&0xFF)},
							},
						},
						{
							Name:            "eth1",
							Flags:           uint32(net.FlagUp),
							HardwareAddress: []byte{2, 3, 4, 5, 6, 7},
							IPAddresses: []MachineNetworkInterfaceIPAddress{
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+3)&0xFF00>>8, (i+3)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+4)&0xFF00>>8, (i+4)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+5)&0xFF00>>8, (i+5)&0xFF)},
							},
						},
						{
							Name:            "eth2",
							Flags:           uint32(net.FlagUp),
							HardwareAddress: []byte{3, 4, 5, 6, 7, 8},
							IPAddresses: []MachineNetworkInterfaceIPAddress{
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+6)&0xFF00>>8, (i+6)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+7)&0xFF00>>8, (i+7)&0xFF)},
							},
						},
						{
							Name:            "eth3",
							Flags:           uint32(net.FlagUp),
							HardwareAddress: []byte{4, 5, 6, 7, 8, 9},
							IPAddresses: []MachineNetworkInterfaceIPAddress{
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+5)&0xFF00>>8, (i+5)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+8)&0xFF00>>8, (i+8)&0xFF)},
								{IPAddress: fmt.Sprintf("192.168.%d.%d", (i+9)&0xFF00>>8, (i+9)&0xFF)},
							},
						},
					},
				}
				err := AddMachine(db, m)
				require.NoError(b, err)

				err = UpsertMachineNetworkInterfaces(db, m.ID, m.MachineNetworkInterfaces...)
				require.NoError(b, err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = GetMachinesByNetworkInterfaceIPAddress(db, "192.168.0.5", MachineRelationNetworkInterfaces)
			}
		})
	}
}
