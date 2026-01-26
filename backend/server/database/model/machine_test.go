package dbmodel

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Sort daemons in the machine by name. It is used by the unit
// test to ensure the predictable order of daemons to validate.
func sortMachineDaemonsByName(machine *Machine) {
	if len(machine.Daemons) == 0 {
		return
	}
	sort.Slice(machine.Daemons, func(i, j int) bool {
		return machine.Daemons[i].Name < machine.Daemons[j].Name
	})
}

// Check if adding machine to database works.
func TestAddMachine(t *testing.T) {
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

	// add another one but with the same address - an error should be raised
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.Contains(t, err.Error(), "duplicate")
}

// Check if updating machine in database works.
func TestUpdateMachine(t *testing.T) {
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

	// Remember the creation time so it can be compared after the update.
	createdAt := m.CreatedAt
	require.NotNil(t, createdAt)

	// change authorization
	m1, err := GetMachineByID(db, m.ID)
	require.NoError(t, err)
	require.False(t, m1.Authorized)

	m.Authorized = true
	// Reset creation time to ensure it is not modified during the update.
	m.CreatedAt = time.Time{}
	err = UpdateMachine(db, m)
	require.NoError(t, err)

	m2, err := GetMachineByID(db, m.ID)
	require.NoError(t, err)
	require.True(t, m2.Authorized)
	require.Equal(t, createdAt, m2.CreatedAt)
}

// Check if getting machine by address.
func TestGetMachineByAddress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// add daemon
	d := NewDaemon(m2, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    1234,
			Key:     "",
		},
	})
	err = AddDaemon(db, d)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
	require.Len(t, m.Daemons, 1)
	require.Len(t, m.Daemons[0].AccessPoints, 1)
	require.Equal(t, AccessPointControl, m.Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", m.Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 1234, m.Daemons[0].AccessPoints[0].Port)
	require.Empty(t, m.Daemons[0].AccessPoints[0].Key)
	require.Equal(t, m, m.Daemons[0].Machine)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	// get deleted machine while do not include deleted machines
	m, err = GetMachineByAddressAndAgentPort(db, "localhost", 8080)
	require.Nil(t, err)
	require.Nil(t, m)
}

// Check if getting machine by its ID.
func TestGetMachineByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing machine
	m, err := GetMachineByID(db, 123)
	require.Nil(t, err)
	require.Nil(t, m)

	// add machine
	m2 := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	// add daemon
	d := NewDaemon(m2, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "dns.example.",
			Port:    953,
			Key:     "abcd",
		},
	})
	d.Version = "1.7.5"
	d.Active = true
	err = AddDaemon(db, d)
	require.NoError(t, err)

	// get added machine
	m, err = GetMachineByID(db, m2.ID)
	require.Nil(t, err)
	require.Equal(t, m2.Address, m.Address)
	require.Len(t, m.Daemons, 1)
	require.Len(t, m.Daemons[0].AccessPoints, 1)
	require.Equal(t, AccessPointControl, m.Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "dns.example.", m.Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 953, m.Daemons[0].AccessPoints[0].Port)
	require.Equal(t, "abcd", m.Daemons[0].AccessPoints[0].Key)
	require.Equal(t, daemonname.DHCPv4, m.Daemons[0].Name)
	require.True(t, m.LastVisitedAt.IsZero())
	require.Equal(t, m, m.Daemons[0].Machine)

	// delete machine
	err = DeleteMachine(db, m)
	require.Nil(t, err)

	m, err = GetMachineByID(db, m2.ID)
	require.NoError(t, err)
	require.Nil(t, m)
}

// Check if getting machine by its ID with relations.
func TestGetMachineByIDWithRelations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:         42,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
		AgentToken: "secret",
	}
	_ = AddMachine(db, m)

	daemonKea := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "dns.example.",
			Port:    953,
			Key:     "abcd",
		},
	})
	err := daemonKea.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"valid-lifetime": 1234,
			"secret": "hidden"
		}
	}`))
	require.NoError(t, err)
	daemonKea.Version = "1.7.5"
	daemonKea.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	err = AddDaemon(db, daemonKea)
	require.NoError(t, err)

	daemonBind9 := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, daemonBind9)
	require.NoError(t, err)

	service := &Service{
		BaseService: BaseService{
			Name: "service",
			Daemons: []*Daemon{
				{
					ID: daemonKea.ID,
				},
			},
		},
		HAService: &BaseHAService{
			HAType:                     "dhcp4",
			Relationship:               "server1",
			PrimaryID:                  daemonKea.ID,
			PrimaryStatusCollectedAt:   time.Now(),
			SecondaryStatusCollectedAt: time.Now(),
			PrimaryReachable:           true,
			SecondaryReachable:         true,
			PrimaryLastState:           "load-balancing",
			SecondaryLastState:         "syncing",
			PrimaryLastScopes:          []string{"server1", "server2"},
			SecondaryLastScopes:        []string{},
			PrimaryLastFailoverAt:      time.Now(),
		},
	}

	err = AddService(db, service)
	require.NoError(t, err)

	// Act
	machine, machineErr := GetMachineByIDWithRelations(db, 42)
	machineDaemons, machineDaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemons)
	machineKeaDaemons, machineKeaDaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationKeaDaemons)
	machineBind9Daemons, machineBind9DaemonsErr := GetMachineByIDWithRelations(db, 42, MachineRelationBind9Daemons)
	machineDaemonLogTargets, machineDaemonLogTargetsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemonLogTargets)
	machineDaemonAccessPoints, machineDaemonAccessPointsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemonAccessPoints)
	machineKeaDHCPConfigs, machineKeaDHCPConfigsErr := GetMachineByIDWithRelations(db, 42, MachineRelationKeaDHCPConfigs)
	machineDaemonAccessPointsKeaDHCPConfigs, machineDaemonAccessPointsKeaDHCPConfigsErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemonAccessPoints, MachineRelationKeaDHCPConfigs)
	machineDaemonHAServices, machineDaemonHAServicesErr := GetMachineByIDWithRelations(db, 42, MachineRelationDaemonHAServices)

	// Assert
	require.NoError(t, machineErr)
	require.NoError(t, machineDaemonsErr)
	require.NoError(t, machineKeaDaemonsErr)
	require.NoError(t, machineBind9DaemonsErr)
	require.NoError(t, machineDaemonLogTargetsErr)
	require.NoError(t, machineDaemonAccessPointsErr)
	require.NoError(t, machineKeaDHCPConfigsErr)
	require.NoError(t, machineDaemonAccessPointsKeaDHCPConfigsErr)
	require.NoError(t, machineDaemonHAServicesErr)

	// Just machine
	require.NotNil(t, machine.State)
	require.Len(t, machine.Daemons, 0)
	// Machine with daemons
	require.Len(t, machineDaemons.Daemons, 2)
	sortMachineDaemonsByName(machineDaemons)
	require.Nil(t, machineDaemons.Daemons[0].LogTargets)
	require.Nil(t, machineDaemons.Daemons[0].KeaDaemon)
	require.Equal(t, machineDaemons, machineDaemons.Daemons[0].Machine)
	require.Nil(t, machineDaemons.Daemons[1].Bind9Daemon)
	require.Equal(t, machineDaemons, machineDaemons.Daemons[1].Machine)
	// Machine with kea daemons
	require.Len(t, machineKeaDaemons.Daemons, 2)
	sortMachineDaemonsByName(machineKeaDaemons)
	require.Nil(t, machineKeaDaemons.Daemons[0].KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, machineKeaDaemons.Daemons[0].LogTargets)
	require.Equal(t, machineKeaDaemons, machineKeaDaemons.Daemons[0].Machine)
	require.Nil(t, machineKeaDaemons.Daemons[1].Bind9Daemon)
	require.Equal(t, machineKeaDaemons, machineKeaDaemons.Daemons[1].Machine)
	// Machine with Bind9 daemons
	require.Len(t, machineBind9Daemons.Daemons, 2)
	sortMachineDaemonsByName(machineBind9Daemons)
	require.Nil(t, machineBind9Daemons.Daemons[0].KeaDaemon)
	require.Nil(t, machineBind9Daemons.Daemons[0].LogTargets)
	require.Equal(t, machineBind9Daemons, machineBind9Daemons.Daemons[0].Machine)
	require.NotNil(t, machineBind9Daemons.Daemons[1].Bind9Daemon)
	require.Equal(t, machineBind9Daemons, machineBind9Daemons.Daemons[1].Machine)
	// Machine with daemon log targets
	require.Len(t, machineDaemonLogTargets.Daemons, 2)
	sortMachineDaemonsByName(machineDaemonLogTargets)
	require.Nil(t, machineDaemonLogTargets.Daemons[0].KeaDaemon)
	require.Len(t, machineDaemonLogTargets.Daemons[0].LogTargets, 2)
	require.Nil(t, machineDaemonLogTargets.Daemons[0].Bind9Daemon)
	require.Equal(t, machineDaemonLogTargets, machineDaemonLogTargets.Daemons[0].Machine)
	// Machine with the access points
	require.Len(t, machineDaemonAccessPoints.Daemons, 2)
	sortMachineDaemonsByName(machineDaemonAccessPoints)
	require.Len(t, machineDaemonAccessPoints.Daemons[0].AccessPoints, 1)
	require.Equal(t, machineDaemonAccessPoints, machineDaemonAccessPoints.Daemons[0].Machine)
	require.Empty(t, machineDaemonAccessPoints.Daemons[1].AccessPoints)
	require.Equal(t, machineDaemonAccessPoints, machineDaemonAccessPoints.Daemons[1].Machine)
	// Machine with Kea DHCP configurations
	require.Len(t, machineKeaDHCPConfigs.Daemons, 2)
	sortMachineDaemonsByName(machineKeaDHCPConfigs)
	require.NotNil(t, machineKeaDHCPConfigs.Daemons[0].KeaDaemon.KeaDHCPDaemon)
	// Machine with the access points and Kea DHCP configurations
	require.Len(t, machineDaemonAccessPointsKeaDHCPConfigs.Daemons, 2)
	sortMachineDaemonsByName(machineDaemonAccessPointsKeaDHCPConfigs)
	require.NotNil(t, machineDaemonAccessPointsKeaDHCPConfigs.Daemons[0].KeaDaemon.KeaDHCPDaemon)
	require.Len(t, machineDaemonAccessPointsKeaDHCPConfigs.Daemons[0].AccessPoints, 1)
	require.Equal(t, machineDaemonAccessPointsKeaDHCPConfigs, machineDaemonAccessPointsKeaDHCPConfigs.Daemons[0].Machine)
	// Machine with the HA services
	require.Len(t, machineDaemonHAServices.Daemons, 2)
	sortMachineDaemonsByName(machineDaemonHAServices)
	require.Len(t, machineDaemonHAServices.Daemons[0].Services, 1)
	require.NotNil(t, machineDaemonHAServices.Daemons[0].Services[0].HAService)
	require.Empty(t, machineDaemonHAServices.Daemons[1].Services)
	require.Equal(t, machineDaemonHAServices, machineDaemonHAServices.Daemons[0].Machine)
	require.Equal(t, machineDaemonHAServices, machineDaemonHAServices.Daemons[1].Machine)
}

// Test that the machine is selected by the address and port of an access point.
func TestGetMachineByAddressAndAccessPointPort(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m1 := &Machine{Address: "fe80::1", AgentPort: 8080}
	_ = AddMachine(db, m1)

	d1 := NewDaemon(m1, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "fe80::1",
			Port:    8001,
		},
	})
	_ = AddDaemon(db, d1)

	d2 := NewDaemon(m1, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "127.0.0.1",
			Port:    8003,
		},
	})
	_ = AddDaemon(db, d2)

	m2 := &Machine{Address: "fe80::1:1", AgentPort: 8090}
	_ = AddMachine(db, m2)

	d3 := NewDaemon(m2, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "fe80::1:2",
			Port:    8001,
		},
	})
	_ = AddDaemon(db, d3)

	// Act
	machine, err := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8001, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, machine)
	require.EqualValues(t, m1.ID, machine.ID)
	require.Len(t, machine.Daemons, 2)
	require.Equal(t, machine, machine.Daemons[0].Machine)
	require.Equal(t, machine, machine.Daemons[1].Machine)
}

// Test that the machine is selected by the address, port, and type of an
// access point.
func TestGetMachineByAddressAndAccessPointPortFilterByType(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m1 := &Machine{Address: "fe80::1", AgentPort: 8080}
	_ = AddMachine(db, m1)

	d1 := NewDaemon(m1, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "127.0.0.1",
			Port:    8001,
		},
	})
	_ = AddDaemon(db, d1)

	d2 := NewDaemon(m1, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointStatistics,
			Address: "127.0.0.1",
			Port:    8003,
		},
	})
	_ = AddDaemon(db, d2)

	t.Run("Filter the Kea Control Agent only", func(t *testing.T) {
		accessPointType := AccessPointControl

		// Act
		machineControl, errControl := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8001, &accessPointType)
		machineStatistics, errStatistics := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8003, &accessPointType)

		// Assert
		require.NoError(t, errControl)
		require.NoError(t, errStatistics)

		require.NotNil(t, machineControl)
		require.Nil(t, machineStatistics)
	})

	t.Run("Filter the Statistics channel only", func(t *testing.T) {
		accessPointType := AccessPointStatistics

		// Act
		machineControl, errControl := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8001, &accessPointType)
		machineStatistics, errStatistics := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8003, &accessPointType)

		// Assert
		require.NoError(t, errControl)
		require.NoError(t, errStatistics)

		require.Nil(t, machineControl)
		require.NotNil(t, machineStatistics)
	})

	t.Run("No type filter", func(t *testing.T) {
		// Act
		machineControl, errControl := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8001, nil)
		machineStatistics, errStatistics := GetMachineByAddressAndAccessPointPort(db, "fe80::1", 8003, nil)

		// Assert
		require.NoError(t, errControl)
		require.NoError(t, errStatistics)

		require.NotNil(t, machineControl)
		require.NotNil(t, machineStatistics)
	})
}

// Basic check if getting machines by pages works.
func TestGetMachinesByPageBasic(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// no machines yet but try to get some
	ms, total, err := GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.Zero(t, total)
	require.Len(t, ms, 0)

	// add 10 machines
	for i := 1; i <= 10; i++ {
		m := &Machine{
			Address:   fmt.Sprintf("host-%d", 21-i),
			AgentPort: int64(i),
		}
		err = AddMachine(db, m)
		require.NoError(t, err)

		// add daemon
		d := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    int64(8000 + i),
				Key:     "",
			},
		})
		err = AddDaemon(db, d)
		require.NoError(t, err)
	}

	// get 10 machines from 0
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.Len(t, ms[5].Daemons, 1)
	require.Equal(t, ms[5].Daemons[0].Machine, &ms[5])

	// get 2 machines out of 10, from 0
	ms, total, err = GetMachinesByPage(db, 0, 2, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 2)

	// get 3 machines out of 10, from 2
	ms, total, err = GetMachinesByPage(db, 2, 3, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 3)

	// get 10 machines out of 10, from 0, but with '2' in contents; should return 1: 20 and 12
	text := "2"
	ms, total, err = GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, ms, 2)

	// check machine details
	require.Len(t, ms[0].Daemons, 1)
	require.Len(t, ms[0].Daemons[0].AccessPoints, 1)
	require.Equal(t, AccessPointControl, ms[0].Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", ms[0].Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 8001, ms[0].Daemons[0].AccessPoints[0].Port)
	require.Empty(t, ms[0].Daemons[0].AccessPoints[0].Key)

	require.Len(t, ms[1].Daemons, 1)
	require.Len(t, ms[1].Daemons[0].AccessPoints, 1)
	require.Equal(t, AccessPointControl, ms[1].Daemons[0].AccessPoints[0].Type)
	require.Equal(t, "localhost", ms[1].Daemons[0].AccessPoints[0].Address)
	require.EqualValues(t, 8009, ms[1].Daemons[0].AccessPoints[0].Port)
	require.Empty(t, ms[1].Daemons[0].AccessPoints[0].Key)

	// check sorting by id asc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "", SortDirAsc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 1, ms[0].ID)
	require.EqualValues(t, 6, ms[5].ID)

	// check sorting by id desc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "", SortDirDesc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 10, ms[0].ID)
	require.EqualValues(t, 5, ms[5].ID)

	// check sorting by address asc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "address", SortDirAsc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 10, ms[0].ID)
	require.EqualValues(t, 5, ms[5].ID)

	// check sorting by address desc
	ms, total, err = GetMachinesByPage(db, 0, 100, nil, nil, "address", SortDirDesc)
	require.Nil(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, ms, 10)
	require.EqualValues(t, 1, ms[0].ID)
	require.EqualValues(t, 6, ms[5].ID)
}

// Check if getting machines with filtering by pages works.
func TestGetMachinesByPageWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
		State: MachineState{
			Hostname:       "my-host",
			PlatformFamily: "redhat",
		},
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// filter machines by json fields: redhat
	text := "redhat"
	ms, total, err := GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)

	// filter machines by json fields: my
	text = "my"
	ms, total, err = GetMachinesByPage(db, 0, 10, &text, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
}

// Check if getting authorized machines with filtering by pages works.
func TestGetMachinesByPageFilteredByAuthorized(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine2
	m := &Machine{
		Address:   "unauthorized",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	m = &Machine{
		Address:    "authorized",
		AgentPort:  8080,
		Authorized: true,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)

	// get unauthorized machines
	authorized := false
	ms, total, err := GetMachinesByPage(db, 0, 10, nil, &authorized, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
	require.EqualValues(t, "unauthorized", ms[0].Address)
	require.False(t, ms[0].Authorized)

	// get authorized machines
	authorized = true
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, &authorized, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, ms, 1)
	require.EqualValues(t, "authorized", ms[0].Address)
	require.True(t, ms[0].Authorized)

	// get all machines
	ms, total, err = GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.Nil(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, ms, 2)
}

// Test getting the number of unauthorized machines.
func TestGetUnauthorizedMachinesCount(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		m := &Machine{
			Address:   fmt.Sprintf("machine%d", i),
			AgentPort: 8080,
		}
		if i > 7 {
			m.Authorized = true
		}
		err := AddMachine(db, m)
		require.NoError(t, err)
	}

	count, err := GetUnauthorizedMachinesCount(db)
	require.NoError(t, err)
	require.EqualValues(t, 8, count)
}

// Check if an attempt to delete a machine without specifying the daemons
// relation fails.
func TestDeleteMachineOnly(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// delete machine
	err = DeleteMachine(db, m)
	require.Error(t, err)
	require.Contains(t, err.Error(), "deleted machine with ID 1 has no daemons relation")
}

// Check if deleting machine and its daemons works.
func TestDeleteMachineWithDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// add daemon
	d := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, d)
	require.NoError(t, err)
	daemonID := d.ID
	require.NotEqual(t, 0, daemonID)

	m.Daemons = []*Daemon{d}

	// reload machine from db to get daemons relation loaded
	err = RefreshMachineFromDB(db, m)
	require.Nil(t, err)
	require.Len(t, m.Daemons, 1)
	require.Equal(t, m, m.Daemons[0].Machine)

	// delete machine
	err = DeleteMachine(db, m)
	require.NoError(t, err)

	// check if daemon is also deleted
	d, err = GetDaemonByID(db, daemonID)
	require.NoError(t, err)
	require.Nil(t, d)
}

// Test that the machine associated with an empty config report may be deleted
// properly.
func TestDeleteMachineWithEmptyConfigReport(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, machine)

	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	err := AddDaemon(db, daemon)
	require.NoError(t, err)

	configReport := &ConfigReport{
		CheckerName: "empty",
		Content:     nil,
		DaemonID:    daemon.ID,
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	machine.Daemons = []*Daemon{daemon}

	// Act
	err = DeleteMachine(db, machine)

	// Assert
	require.NoError(t, err)
}

// Test that the machine associated with a config report may be deleted
// properly.
func TestDeleteMachineWithConfigReport(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, machine)

	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	err := AddDaemon(db, daemon)
	require.NoError(t, err)

	configReport := &ConfigReport{
		CheckerName: "checker",
		Content:     newPtr("my {daemon}"),
		DaemonID:    daemon.ID,
		RefDaemons:  []*Daemon{daemon},
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	machine.Daemons = []*Daemon{daemon}

	// Act
	err = DeleteMachine(db, machine)

	// Assert
	require.NoError(t, err)
}

// Test deleting a machine and cascaded deletion of the orphaned
// objects such as subnets, hosts and shared networks.
func TestDeleteMachineWithKeaDaemonOrphans(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// Add a daemon.
	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	m, err = GetMachineByID(db, m.ID)
	require.NoError(t, err)

	// Add shared network.
	sharedNetwork := &SharedNetwork{
		Name:   "my-shared-network",
		Family: 4,
	}
	err = AddSharedNetwork(db, sharedNetwork)
	require.NoError(t, err)

	// Add subnet.
	subnet := &Subnet{
		Prefix:          "192.0.2.0/24",
		SharedNetworkID: sharedNetwork.ID,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: daemon.ID,
			},
		},
	}
	err = AddSubnet(db, subnet)
	require.NoError(t, err)

	// Add host.
	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
		},
		LocalHosts: []LocalHost{
			{
				DataSource: HostDataSourceAPI,
				DaemonID:   daemon.ID,
			},
		},
	}
	err = AddHost(db, host)
	require.NoError(t, err)

	// Deleting the machine should cause deletion of the associated shared
	// networks, subnets and hosts.
	err = DeleteMachine(db, m)
	require.NoError(t, err)

	returnedSharedNetworks, err := GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Empty(t, returnedSharedNetworks)

	returnedSubnets, err := GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)

	returnedHosts, err := GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Empty(t, returnedHosts)
}

// Test deleting a machine and cascaded deletion of the orphaned
// objects such as zones.
func TestDeleteMachineWithDNSDaemonsOrphans(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, machine)

	// Add daemons.
	bind9Daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{})
	_ = AddDaemon(db, bind9Daemon)
	pdnsDaemon := NewDaemon(machine, daemonname.PDNS, true, []*AccessPoint{})
	_ = AddDaemon(db, pdnsDaemon)

	// Re-fetch the machine instance.
	machine, _ = GetMachineByID(db, machine.ID)

	// Add zones.
	zone := &Zone{
		Name: "exampleA.org",
		LocalZones: []*LocalZone{
			{
				DaemonID: bind9Daemon.ID,
				Class:    "IN",
				Type:     "master",
				Serial:   1,
				LoadedAt: time.Now(),
			},
		},
	}
	_ = AddZones(db, zone)

	zone = &Zone{
		Name: "exampleB.com",
		LocalZones: []*LocalZone{
			{
				DaemonID: pdnsDaemon.ID,
				Class:    "IN",
				Type:     "master",
				Serial:   1,
				LoadedAt: time.Now(),
			},
		},
	}
	_ = AddZones(db, zone)

	// Act
	// Deleting the machine should cause deletion of the associated zone.
	err := DeleteMachine(db, machine)

	// Assert
	require.NoError(t, err)

	returnedZones, _, err := GetZones(db, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Empty(t, returnedZones)
}

// Check if refreshing machine works.
func TestRefreshMachineFromDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
		Error:     "some error",
		State: MachineState{
			Hostname: "aaaa",
			Cpus:     4,
		},
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	m.State.Hostname = "bbbb"
	m.State.Cpus = 2
	m.Error = ""

	err = RefreshMachineFromDB(db, m)
	require.Nil(t, err)
	require.Equal(t, "aaaa", m.State.Hostname)
	require.EqualValues(t, 4, m.State.Cpus)
	require.Equal(t, "some error", m.Error)
}

// Check if getting all machines works.
func TestGetAllMachines(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add 20 machines
	for i := 1; i <= 20; i++ {
		m := &Machine{
			Address:   "localhost",
			AgentPort: 8080 + int64(i),
			Error:     "some error",
			State: MachineState{
				Hostname: "aaaa",
				Cpus:     4,
			},
			Authorized: i%2 == 0,
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		d := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    1234,
				Key:     "",
			},
		})
		err = AddDaemon(db, d)
		require.NoError(t, err)

		cr := &ConfigReview{
			ConfigHash: "1234",
			Signature:  "2345",
			DaemonID:   d.ID,
		}
		err = AddConfigReview(db, cr)
		require.NoError(t, err)
	}

	// get all machines should return 20 machines
	machines, err := GetAllMachines(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 20)
	require.EqualValues(t, "localhost", machines[0].Address)
	require.EqualValues(t, "localhost", machines[19].Address)
	require.EqualValues(t, "some error", machines[0].Error)
	require.EqualValues(t, "some error", machines[19].Error)
	require.EqualValues(t, 4, machines[0].State.Cpus)
	require.EqualValues(t, 4, machines[19].State.Cpus)
	require.NotEqual(t, machines[0].AgentPort, machines[19].AgentPort)
	require.Len(t, machines[10].Daemons, 1)
	require.Equal(t, &machines[10], machines[10].Daemons[0].Machine)

	// Ensure that we fetched daemons and config reviews too.
	require.Len(t, machines[0].Daemons, 1)
	require.NotNil(t, machines[0].Daemons[0].ConfigReview)
	require.Equal(t, "1234", machines[0].Daemons[0].ConfigReview.ConfigHash)
	require.Equal(t, "2345", machines[0].Daemons[0].ConfigReview.Signature)

	// get only unauthorized machines
	authorized := false
	machines, err = GetAllMachines(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)

	// and now only authorized machines
	authorized = true
	machines, err = GetAllMachines(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)

	// paged get should return indicated limit, not all
	machines, total, err := GetMachinesByPage(db, 0, 10, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, machines, 10)
	require.EqualValues(t, 20, total)
}

// Test that getting all machines with relations works.
func TestGetAllMachinesWithRelations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = AddMachine(db, m)

	mUnauthorized := &Machine{
		Address:    "unauthorized",
		AgentPort:  8081,
		Authorized: false,
	}
	_ = AddMachine(db, mUnauthorized)

	daemonKea := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{{
		Type:    AccessPointControl,
		Address: "kea",
		Port:    8000,
		Key:     "",
	}})
	daemonKea.LogTargets = append(daemonKea.LogTargets, &LogTarget{
		Name:     "syslog",
		Severity: "info",
		Output:   "syslog",
	})
	_ = AddDaemon(db, daemonKea)
	_ = AddConfigReview(db, &ConfigReview{
		ConfigHash: "hash",
		Signature:  "signature",
		DaemonID:   daemonKea.ID,
	})

	daemonBind9 := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{{
		Type:    AccessPointControl,
		Address: "bind9",
		Port:    8053,
		Key:     "",
	}})
	_ = AddDaemon(db, daemonBind9)

	daemonPDNS := NewDaemon(m, daemonname.PDNS, true, []*AccessPoint{{
		Type:    AccessPointControl,
		Address: "pdns",
		Port:    8054,
		Key:     "",
	}})
	_ = AddDaemon(db, daemonPDNS)

	_ = AddService(db, &Service{
		BaseService: BaseService{
			Name:    "service",
			Daemons: []*Daemon{daemonKea},
		},
		HAService: &BaseHAService{
			HAType:       daemonname.DHCPv4,
			HAMode:       HAModeHotStandby,
			Relationship: "server1",
			PrimaryID:    daemonKea.ID,
		},
	})

	t.Run("No relations - all", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, nil)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 2)
		require.Nil(t, machines[0].Daemons)
	})

	t.Run("No relations - unauthorized", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(false))

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Equal(t, "unauthorized", machines[0].Address)
	})

	t.Run("No relations - authorized", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true))

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Equal(t, "localhost", machines[0].Address)
	})

	t.Run("With daemons", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationDaemons,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, &machines[0], machines[0].Daemons[0].Machine)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
	})

	t.Run("With log targets", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationDaemonLogTargets,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Len(t, machines[0].Daemons[0].LogTargets, 1)
		require.Equal(t, "syslog", machines[0].Daemons[0].LogTargets[0].Name)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With access points", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationDaemonAccessPoints,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Len(t, machines[0].Daemons[0].AccessPoints, 1)
		require.Equal(t, "kea", machines[0].Daemons[0].AccessPoints[0].Address)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Len(t, machines[0].Daemons[1].AccessPoints, 1)
		require.Equal(t, "bind9", machines[0].Daemons[1].AccessPoints[0].Address)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Len(t, machines[0].Daemons[2].AccessPoints, 1)
		require.Equal(t, "pdns", machines[0].Daemons[2].AccessPoints[0].Address)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With Kea daemons", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationKeaDaemons,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.NotNil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon.KeaDHCPDaemon)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With BIND9 daemons", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationBind9Daemons,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.NotNil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With PDNS daemons", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationPDNSDaemons,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.NotNil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With Kea DHCP configs", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationKeaDHCPConfigs,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.NotNil(t, machines[0].Daemons[0].KeaDaemon)
		require.NotNil(t, machines[0].Daemons[0].KeaDaemon.KeaDHCPDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With config reviews", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationDaemonConfigReview,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.NotNil(t, machines[0].Daemons[0].ConfigReview)
		require.Equal(t, "hash", machines[0].Daemons[0].ConfigReview.ConfigHash)
		require.Equal(t, "signature", machines[0].Daemons[0].ConfigReview.Signature)
		require.Empty(t, machines[0].Daemons[0].Services)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With services", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationDaemonHAServices,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Empty(t, machines[0].Daemons[0].LogTargets)
		require.Empty(t, machines[0].Daemons[0].AccessPoints)
		require.Nil(t, machines[0].Daemons[0].KeaDaemon)
		require.Nil(t, machines[0].Daemons[0].ConfigReview)
		require.Len(t, machines[0].Daemons[0].Services, 1)
		require.Equal(t, "service", machines[0].Daemons[0].Services[0].Name)
		require.NotNil(t, machines[0].Daemons[0].Services[0].HAService)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.Nil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.Nil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})

	t.Run("With all relations", func(t *testing.T) {
		// Act
		machines, err := GetAllMachinesWithRelations(db, storkutil.Ptr(true),
			MachineRelationBind9Daemons,
			MachineRelationDaemonAccessPoints,
			MachineRelationDaemonConfigReview,
			MachineRelationDaemonHAServices,
			MachineRelationKeaDaemons,
			MachineRelationKeaDHCPConfigs,
			MachineRelationPDNSDaemons,
			MachineRelationDaemonLogTargets,
		)

		// Assert
		require.NoError(t, err)
		require.Len(t, machines, 1)
		require.Len(t, machines[0].Daemons, 3)
		require.Equal(t, daemonname.DHCPv4, machines[0].Daemons[0].Name)
		require.Len(t, machines[0].Daemons[0].LogTargets, 1)
		require.Len(t, machines[0].Daemons[0].AccessPoints, 1)
		require.NotNil(t, machines[0].Daemons[0].KeaDaemon)
		require.NotNil(t, machines[0].Daemons[0].KeaDaemon.KeaDHCPDaemon)
		require.NotNil(t, machines[0].Daemons[0].ConfigReview)
		require.Len(t, machines[0].Daemons[0].Services, 1)
		require.Equal(t, "service", machines[0].Daemons[0].Services[0].Name)
		require.Equal(t, daemonname.Bind9, machines[0].Daemons[1].Name)
		require.NotNil(t, machines[0].Daemons[1].Bind9Daemon)
		require.Len(t, machines[0].Daemons[1].AccessPoints, 1)
		require.Nil(t, machines[0].Daemons[1].ConfigReview)
		require.Empty(t, machines[0].Daemons[1].Services)
		require.Equal(t, daemonname.PDNS, machines[0].Daemons[2].Name)
		require.NotNil(t, machines[0].Daemons[2].PDNSDaemon)
		require.Len(t, machines[0].Daemons[2].AccessPoints, 1)
		require.Nil(t, machines[0].Daemons[2].ConfigReview)
		require.Empty(t, machines[0].Daemons[2].Services)
	})
}

// Test MachineTag interface implementation.
func TestMachineTag(t *testing.T) {
	machine := Machine{
		ID:        10,
		Address:   "192.0.2.2",
		AgentPort: 1234,
		State: MachineState{
			Hostname: "cool.example.org",
		},
	}
	require.EqualValues(t, 10, machine.GetID())
	require.Equal(t, "192.0.2.2", machine.GetAddress())
	require.EqualValues(t, 1234, machine.GetAgentPort())
	require.Equal(t, "cool.example.org", machine.GetHostname())
}

// Check if getting all machines without any relations works.
func TestGetAllMachinesNoRelations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add 20 machines
	for i := 1; i <= 20; i++ {
		m := &Machine{
			Address:   "localhost",
			AgentPort: 8080 + int64(i),
			Error:     "some error",
			State: MachineState{
				Hostname: "aaaa",
				Cpus:     4,
			},
			Authorized: i%2 == 0,
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		d := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    1234,
				Key:     "",
			},
		})
		err = AddDaemon(db, d)
		require.NoError(t, err)

		cr := &ConfigReview{
			ConfigHash: "1234",
			Signature:  "2345",
			DaemonID:   d.ID,
		}
		err = AddConfigReview(db, cr)
		require.NoError(t, err)
	}

	// get all machines should return 20 machines
	machines, err := GetAllMachinesNoRelations(db, nil)
	require.NoError(t, err)
	require.Len(t, machines, 20)

	for i, machine := range machines {
		require.EqualValues(t, "localhost", machine.Address)
		require.EqualValues(t, "some error", machine.Error)
		require.EqualValues(t, 4, machine.State.Cpus)
		if i > 0 {
			require.NotEqual(t, machines[i-1].AgentPort, machine.AgentPort)
		}
		// Ensure that no relations were involved.
		require.Nil(t, machine.Daemons)
	}

	// get only unauthorized machines
	authorized := false
	machines, err = GetAllMachinesNoRelations(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)
	for _, machine := range machines {
		require.False(t, machine.Authorized)
	}

	// and now only authorized machines
	authorized = true
	machines, err = GetAllMachinesNoRelations(db, &authorized)
	require.NoError(t, err)
	require.Len(t, machines, 10)
	for _, machine := range machines {
		require.True(t, machine.Authorized)
	}
}

// Test GetLabel method returns hostname when available.
func TestGetLabelWithHostname(t *testing.T) {
	machine := Machine{
		Address: "2001:db8::1",
		State: MachineState{
			Hostname: "myhost.example.org",
		},
	}
	require.Equal(t, "myhost.example.org", machine.GetLabel())
}

// Test GetLabel method returns address when hostname is empty.
func TestGetLabelWithoutHostname(t *testing.T) {
	machine := Machine{
		Address: "2001:db8::1",
		State:   MachineState{},
	}
	require.Equal(t, "2001:db8::1", machine.GetLabel())
}
