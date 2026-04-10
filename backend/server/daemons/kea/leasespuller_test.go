package kea

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	agentapi "isc.org/stork/api"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=kea -destination=connectedagentsmock_test.go -source=../../agentcomm/agentcomm.go ConnectedAgents

// Return a fake daemon that can be filtered by filterDaemons in the tests below.
func mockFilterableDaemon(name daemonname.Name, id, machineID int64) *dbmodel.Daemon {
	return &dbmodel.Daemon{
		Name:      name,
		ID:        id,
		MachineID: machineID,
	}
}

// Accept a daemon and a lease database, and return a daemon configured with
// that lease database. Callers should not rely on the structures entering and
// leaving this function to be at the same location in memory.
func daemonWithLeaseDB(d *dbmodel.Daemon, ldb *keaconfig.Database) *dbmodel.Daemon {
	// Misery.
	switch d.Name {
	case daemonname.DHCPv4:
		d.KeaDaemon = &dbmodel.KeaDaemon{
			Config: &dbmodel.KeaConfig{
				Config: &keaconfig.Config{
					DHCPv4Config: &keaconfig.DHCPv4Config{
						CommonDHCPConfig: keaconfig.CommonDHCPConfig{
							LeaseDatabase: ldb,
						},
					},
				},
			},
		}
	case daemonname.DHCPv6:
		d.KeaDaemon = &dbmodel.KeaDaemon{
			Config: &dbmodel.KeaConfig{
				Config: &keaconfig.Config{
					DHCPv6Config: &keaconfig.DHCPv6Config{
						CommonDHCPConfig: keaconfig.CommonDHCPConfig{
							LeaseDatabase: ldb,
						},
					},
				},
			},
		}
	default:
		// Do nothing
	}
	return d
}

// Accept a daemon and a memfile path/persist setting and return a daemon
// configured with that path as a memfile lease database. Callers should not
// rely on the structures entering and leaving this function to be at the same
// location in memory.
func daemonWithLeaseMemfile(d *dbmodel.Daemon, path string, persist *bool) *dbmodel.Daemon {
	ldb := keaconfig.Database{
		Type:    "memfile",
		Path:    path,
		Persist: persist,
	}
	return daemonWithLeaseDB(d, &ldb)
}

// Accept a daemon and some SQL connection settings, and return a daemon
// configured with that SQL database as the memfile lease database. Callers
// should not rely on the structures entering and leaving this function to be at
// the same location in memory.
func daemonWithLeaseSQLDB(d *dbmodel.Daemon, kind, host string) *dbmodel.Daemon {
	ldb := keaconfig.Database{
		Type: kind,
		Host: host,
	}
	return daemonWithLeaseDB(d, &ldb)
}

// Accept a daemon and the IDs of two other daemons, and return a daemon
// configured to be in a triangle of high availability partner relationships.
// Each peer is the primary for one peer and the secondary for the other peer.
// Pay close attention to which ID you put in which parameter: this daemon is
// the standby for the relationship with the peer ID in standbyFor, and it is
// the primary for the relationship with the peer ID in primaryFor. Callers
// should not rely on the structures entering and leaving this function to be at
// the same location in memory.
func daemonWithHAHotStandbyTriangle(t *testing.T, d *dbmodel.Daemon, standbyFor, primaryFor int64) *dbmodel.Daemon {
	myID := d.ID
	hooks := keaconfig.HookLibraries{
		{
			Library: "/usr/lib/kea/libdhcp_ha.so",
			Parameters: (json.RawMessage)(fmt.Sprintf(`{
				"high-availability": [
					{
						"this-server-name": "server%[1]d",
						"mode": "hot-standby",
						"heartbeat-delay": 10000,
						"peers": [{
							"name": "server%[1]d",
							"url": "http://192.168.1.%[1]d:8005/",
							"role": "primary",
							"auto-failover": true
						}, {
							"name": "server%[2]d",
							"url": "http://192.168.1.%[2]d:8005/",
							"role": "standby",
							"auto-failover": true
						}]
					},
					{
						"this-server-name": "server%[1]d",
						"mode": "hot-standby",
						"heartbeat-delay": 10000,
						"peers": [{
							"name": "server%[3]d",
							"url": "http://192.168.1.%[3]d:8005/",
							"role": "primary",
							"auto-failover": true
						}, {
							"name": "server%[1]d",
							"url": "http://192.168.1.%[1]d:8005/",
							"role": "standby",
							"auto-failover": true
						}]
					}
				]
			}`,
				myID,
				primaryFor,
				standbyFor,
			)),
		},
	}
	switch d.Name {
	case daemonname.DHCPv4:
		d.KeaDaemon.Config.DHCPv4Config.HookLibraries = hooks
	case daemonname.DHCPv6:
		d.KeaDaemon.Config.DHCPv6Config.HookLibraries = hooks
	default:
		require.FailNow(t, "test function used incorrectly, must pass a DHCPv4 or DHCPv6 daemon.")
	}
	return d
}

// Sort the provided slice of daemons by their daemon IDs, in ascending order.
func sortByID(daemons []*dbmodel.Daemon) []*dbmodel.Daemon {
	slices.SortStableFunc(daemons, func(a, b *dbmodel.Daemon) int {
		return int(a.ID - b.ID)
	})
	return daemons
}

// Test a variety of inputs to checkIsLocalhost to ensure that it accurately
// detects localhost URLs in a variety of normal and edge cases.
func TestCheckIsLocalhost(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input    string
		expected bool
	}{
		{"127.0.0.1", true},              // plain IPv4 loopback
		{"127.0.0.1:5432", true},         // IPv4 loopback with port
		{"127.0.33.10", true},            // weird IPv4 in loopback subnet
		{"::1", true},                    // plain IPv6 loopback
		{"[::1]:80", true},               // IPv6 loopback in brackets with port
		{"192.168.1.108", false},         // non-loopback IPv4 (private subnet)
		{"8.8.8.8", false},               // non-loopback IPv4 (public subnet)
		{"localhost:5432", true},         // localhost hostname with port
		{"localhost", true},              // localhost hostname without port
		{"postgres.example:5432", false}, // non-locahost hostname with port
		{"postgres.example", false},      // non-locahost hostname without port
		{"localhost.example", false},     // confusing subdomain
	}
	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			t.Parallel()
			result := checkIsLocalhost(testCase.input)
			require.Equal(t, testCase.expected, result)
		})
	}
}

// Test that the leases puller instance is created properly.
func TestNewLeasesPuller(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fa := NewMockConnectedAgents(ctrl)

	// Act
	puller, err := NewLeasesPuller(db, fa)
	defer puller.Shutdown()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, puller)
	require.Equal(t, fa, puller.Agents)
}

// Verify that filterDaemons correctly filters out duplicate daemons (pointing
// to the same database) in a variety of conditions.
func TestFilterDaemons(t *testing.T) {
	var (
		falsePtr                = false
		dbPg                    = "postgresql"
		dbMysql                 = "mysql"
		dbFoo                   = "foo.example:5432"
		dbBar                   = "bar.example:3306"
		dbLocalhostV4           = "127.0.10.33:5432"
		configuredLeasefilePath = "/opt/kea/lease-dhcp4.csv"
		// Daemons configured to use a memfile lease DB:
		memfileDaemonAOnMachine1 = daemonWithLeaseMemfile(
			mockFilterableDaemon(daemonname.DHCPv4, 10, 1),
			configuredLeasefilePath,
			nil,
		)
		memfileDaemonBOnMachine2 = daemonWithLeaseMemfile(
			mockFilterableDaemon(daemonname.DHCPv4, 11, 2),
			configuredLeasefilePath,
			nil,
		)
		memfileDaemonCOnMachine1 = daemonWithLeaseMemfile(
			mockFilterableDaemon(daemonname.DHCPv4, 12, 1),
			configuredLeasefilePath,
			nil,
		)
		// Daemons configured to use a PostgreSQL lease DB:
		sqlDaemonDOnMachine3UsingDBFoo = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv6, 13, 3),
			dbPg,
			dbFoo,
		)
		sqlDaemonEOnMachine4UsingDBFoo = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv6, 14, 4),
			dbPg,
			dbFoo,
		)
		sqlDaemonFOnMachine5UsingDBFoo = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv6, 15, 5),
			dbPg,
			dbFoo,
		)
		// Daemons configured to use a MySQL lease DB:
		sqlDaemon10onMachine3UsingDBBar = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv4, 16, 3),
			dbMysql,
			dbBar,
		)
		sqlDaemon11onMachine4UsingDBBar = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv4, 17, 4),
			dbMysql,
			dbBar,
		)
		memfileDaemon12OnMachine6NoPersist = daemonWithLeaseMemfile(
			mockFilterableDaemon(daemonname.DHCPv4, 18, 6),
			"",
			&falsePtr,
		)
		memfileDaemon13OnMachine6NoPersist = daemonWithLeaseMemfile(
			mockFilterableDaemon(daemonname.DHCPv4, 19, 6),
			"",
			&falsePtr,
		)
		// Daemons in a triangle of HA relationships:
		haDaemon14OnMachine7 = daemonWithHAHotStandbyTriangle(
			t,
			daemonWithLeaseMemfile(
				mockFilterableDaemon(daemonname.DHCPv6, 20, 7),
				configuredLeasefilePath,
				nil,
			),
			22,
			21,
		)
		haDaemon15OnMachine8 = daemonWithHAHotStandbyTriangle(
			t,
			daemonWithLeaseMemfile(
				mockFilterableDaemon(daemonname.DHCPv6, 21, 8),
				configuredLeasefilePath,
				nil,
			),
			20,
			22,
		)
		haDaemon16OnMachine9 = daemonWithHAHotStandbyTriangle(
			t,
			daemonWithLeaseMemfile(
				mockFilterableDaemon(daemonname.DHCPv6, 22, 9),
				configuredLeasefilePath,
				nil,
			),
			21,
			20,
		)
		// Daemons using SQL databases on loopback addresses:
		sqlDaemon17OnMachine10UsingLoopbackDB = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv4, 23, 10),
			dbPg,
			dbLocalhostV4,
		)
		sqlDaemon18OnMachine11UsingLoopbackDB = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv4, 24, 11),
			dbPg,
			dbLocalhostV4,
		)
		sqlDaemon19OnMachine10UsingLoopbackDB = daemonWithLeaseSQLDB(
			mockFilterableDaemon(daemonname.DHCPv4, 25, 10),
			dbPg,
			dbLocalhostV4,
		)
		noLeaseDB = daemonWithLeaseDB(
			mockFilterableDaemon(daemonname.DHCPv6, 26, 12),
			nil,
		)
		// Non-DHCP daemons:
		bind9Daemon = mockFilterableDaemon(daemonname.Bind9, 100, 100)
		keaCADaemon = mockFilterableDaemon(daemonname.CA, 101, 101)
		keaD2Daemon = mockFilterableDaemon(daemonname.D2, 102, 102)
		pdnsDaemon  = mockFilterableDaemon(daemonname.PDNS, 103, 103)
	)
	t.Parallel()
	// Test that one very boring memfile daemon passes through the filter unassailed.
	t.Run("one daemon in, one daemon out", func(t *testing.T) {
		t.Parallel()
		one := []dbmodel.Daemon{
			*memfileDaemonAOnMachine1,
		}
		result := filterDaemons(one, false)

		// Assert
		require.Len(t, result, 1)
		require.Equal(t, memfileDaemonAOnMachine1.ID, result[0].ID)
	})
	// Test that when three daemons pointing to the same SQL DB go into the
	// filter, only the one with the lowest ID comes out.
	t.Run("three daemons pointing to same SQL DB in, lower ID out", func(t *testing.T) {
		t.Parallel()
		three := []dbmodel.Daemon{
			*sqlDaemonDOnMachine3UsingDBFoo,
			*sqlDaemonEOnMachine4UsingDBFoo,
			*sqlDaemonFOnMachine5UsingDBFoo,
		}
		result := filterDaemons(three, false)

		// Assert
		require.Len(t, result, 1)
		require.Equal(t, sqlDaemonDOnMachine3UsingDBFoo.ID, result[0].ID)
	})
	// Test that when two daemons on the same machine pointing to the same memfile
	// on disk go into the filter, only the one with the lowest ID comes out.
	t.Run("two daemons pointing to same memfile in, lower ID out", func(t *testing.T) {
		t.Parallel()
		two := []dbmodel.Daemon{
			*memfileDaemonAOnMachine1,
			*memfileDaemonCOnMachine1,
		}
		result := filterDaemons(two, false)

		// Assert
		require.Len(t, result, 1)
		require.Equal(t, memfileDaemonAOnMachine1.ID, result[0].ID)
	})
	// Test that when a pair of daemons pointed at one SQL DB and a different pair
	// of daemons pointed at a different SQL DB go into the filter, the member of
	// each pair with the lowest ID is returned.
	t.Run("four daemons in (pairs, each pair shares an SQL DB), lower IDs out", func(t *testing.T) {
		t.Parallel()
		four := []dbmodel.Daemon{
			*sqlDaemonDOnMachine3UsingDBFoo,
			*sqlDaemonEOnMachine4UsingDBFoo,
			*sqlDaemon10onMachine3UsingDBBar,
			*sqlDaemon11onMachine4UsingDBBar,
		}
		result := filterDaemons(four, false)

		result = sortByID(result)

		// Assert
		require.Len(t, result, 2)
		require.Equal(t, sqlDaemonDOnMachine3UsingDBFoo.ID, result[0].ID)
		require.Equal(t, sqlDaemon10onMachine3UsingDBBar.ID, result[1].ID)
	})
	// Test that two memfile daemons on different machines both make it through
	// the filter.
	t.Run("two daemons in, two daemons out", func(t *testing.T) {
		t.Parallel()
		two := []dbmodel.Daemon{
			*memfileDaemonAOnMachine1,
			*memfileDaemonBOnMachine2,
		}
		result := filterDaemons(two, false)

		result = sortByID(result)

		// Assert
		require.Len(t, result, 2)
		require.Equal(t, memfileDaemonAOnMachine1.ID, result[0].ID)
		require.Equal(t, memfileDaemonBOnMachine2.ID, result[1].ID)
	})
	// Test that in an HA triangle where each daemon uses a separate memfile, all
	// three machines are returned (this is important for verifying that HA peers
	// all have the same view of the lease database, which is a thing people want
	// to know after configuring HA).
	t.Run("HA triangle with each peer using a separate memfile in, all three out", func(t *testing.T) {
		t.Parallel()
		three := []dbmodel.Daemon{
			*haDaemon14OnMachine7,
			*haDaemon15OnMachine8,
			*haDaemon16OnMachine9,
		}
		result := filterDaemons(three, false)

		result = sortByID(result)

		// Assert
		require.Len(t, result, 3)
		require.Equal(t, haDaemon14OnMachine7.ID, result[0].ID)
		require.Equal(t, haDaemon15OnMachine8.ID, result[1].ID)
		require.Equal(t, haDaemon16OnMachine9.ID, result[2].ID)
	})
	// Test that putting an empty list in results in an empty list coming out (and
	// not a panic or something).
	t.Run("empty list in, empty list out", func(t *testing.T) {
		t.Parallel()
		none := []dbmodel.Daemon{}
		result := filterDaemons(none, false)
		require.Empty(t, result)
	})
	// Test that any non-DHCP daemons which go into the filter are removed, since
	// it is not possible to get leases from them.
	t.Run("non-DHCP daemons excluded", func(t *testing.T) {
		t.Parallel()
		nondhcp := []dbmodel.Daemon{
			*bind9Daemon,
			*keaCADaemon,
			*keaD2Daemon,
			*pdnsDaemon,
		}
		result := filterDaemons(nondhcp, false)
		require.Empty(t, result)
	})
	// Confirm that the filter sees memfile daemons with `nopersist` set as
	// distinct from each other, even on the same machine.
	t.Run("nopersist daemons are different from each other", func(t *testing.T) {
		t.Parallel()
		two := []dbmodel.Daemon{
			*memfileDaemon12OnMachine6NoPersist,
			*memfileDaemon13OnMachine6NoPersist,
		}
		result := filterDaemons(two, false)

		result = sortByID(result)

		// Assert
		require.Len(t, result, 2)
		require.Equal(t, memfileDaemon12OnMachine6NoPersist.ID, result[0].ID)
		require.Equal(t, memfileDaemon13OnMachine6NoPersist.ID, result[1].ID)
	})
	// Confirm that when the onlyMemfile parameter is set to `true`, the filter
	// also excludes daemons configured to use an SQL lease database. (I need this
	// for development until I get around to adding support for SQL lease
	// databases.)
	t.Run("onlyMemfile excludes SQL daemons", func(t *testing.T) {
		t.Parallel()
		mixed := []dbmodel.Daemon{
			*memfileDaemonAOnMachine1,
			*sqlDaemonDOnMachine3UsingDBFoo,
			*sqlDaemon10onMachine3UsingDBBar,
			*memfileDaemon12OnMachine6NoPersist,
		}

		result := filterDaemons(mixed, true)
		result = sortByID(result)

		// Assert
		require.Len(t, result, 2)
		require.Equal(t, memfileDaemonAOnMachine1.ID, result[0].ID)
		require.Equal(t, memfileDaemon12OnMachine6NoPersist.ID, result[1].ID)
	})
	t.Run("daemons using SQL lease databases on localhost on different machines are different from each other", func(t *testing.T) {
		t.Parallel()
		locahosts := []dbmodel.Daemon{
			*sqlDaemon17OnMachine10UsingLoopbackDB,
			*sqlDaemon18OnMachine11UsingLoopbackDB,
		}

		result := filterDaemons(locahosts, false)
		result = sortByID(result)

		// Assert
		require.Len(t, result, 2)
		require.Equal(t, sqlDaemon17OnMachine10UsingLoopbackDB.ID, result[0].ID)
		require.Equal(t, sqlDaemon18OnMachine11UsingLoopbackDB.ID, result[1].ID)
	})
	t.Run("daemons using SQL lease databases on localhost on the same machine are the same as each other", func(t *testing.T) {
		t.Parallel()
		locahosts := []dbmodel.Daemon{
			*sqlDaemon17OnMachine10UsingLoopbackDB,
			*sqlDaemon19OnMachine10UsingLoopbackDB,
		}

		result := filterDaemons(locahosts, false)

		// Assert
		require.Len(t, result, 1)
		require.Equal(t, sqlDaemon17OnMachine10UsingLoopbackDB.ID, result[0].ID)
	})
	t.Run("daemons without a lease database are skipped", func(t *testing.T) {
		t.Parallel()
		locahosts := []dbmodel.Daemon{
			*noLeaseDB,
		}

		result := filterDaemons(locahosts, false)

		// Assert
		require.Len(t, result, 0)
	})
}

// Test that the lease puller can pull leases from a daemon in the simplest
// possible happy-path case.
func TestLeasePullingBasic(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemon, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemon.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeases := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{Lease: &agentapi.Lease{
				Family:        4,
				IpAddress:     "192.0.2.1",
				Cltt:          42,
				ValidLifetime: 420,
			}},
		},
		[]error{nil},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.pullLeases()

	// Assert
	require.NoError(t, err)
}

// Test that the lease puller collects all errors from failed daemons.
func TestPullLeasesCollectsAllErrors(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemon4, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemon4.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)

	daemon6, _ := dbmodeltest.NewKeaDHCPv6Server(db)
	err = daemon6.Configure(`{ "Dhcp6": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases6.csv"
    	}
	}}`)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	mockLeases4 := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{nil},
		[]error{errors.New("four")},
	)
	mockLeases6 := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{nil},
		[]error{errors.New("six")},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases4)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases6)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.pullLeases()

	// Assert
	require.ErrorContains(t, err, "four")
	require.ErrorContains(t, err, "six")
}

// Test to ensure that getLeasesFromDaemon doesn't try to get leases from an
// inactive daemon.
func TestGetLeasesFromDaemonWhenInactive(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)
	daemon.Active = false

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fa := NewMockConnectedAgents(ctrl)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.getLeasesFromDaemon(daemon)

	// Assert
	require.NoError(t, err)
}

// Test that getLeasesFromDaemon skips Kea daemons that are not DHCPv4 or DHCPv6.
func TestGetLeasesFromDaemonWhenNotDHCPD(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	caDaemonServer, _ := dbmodeltest.NewKeaCAServer(db)
	err := caDaemonServer.Configure(`{"Control-agent": {
		"http-host": "127.0.0.1",
		"http-port": 8000,
		"control-sockets": {
			"dhcp4": {
				"socket-type": "unix",
				"socket-name": "/var/run/kea/dhcp-v4.sock"
			},
			"dhcp6": {
				"socket-type": "unix",
				"socket-name": "/var/run/kea/dhcp-v6.sock"
			}
		},
		"loggers": [
			{ "name": "kea-ctrl-agent", "severity": "INFO" }
		]
	}}`)
	require.NoError(t, err)
	caDaemon, err := caDaemonServer.GetDaemon()
	require.NoError(t, err)

	d2DaemonServer, _ := dbmodeltest.NewKeaD2Server(db)
	err = d2DaemonServer.Configure(`{ "DhcpDdns": {
		"ip-address": "127.0.0.1",
		"port": 53001,
		"dns-server-timeout": 500,
		"ncr-protocol": "UDP",
		"ncr-format": "JSON",
		"tsig-keys": [],
		"forward-ddns": {
			"ddns-domains": []
		},
		"reverse-dns": {
			"ddns-domains": []
		}
	}}`)
	require.NoError(t, err)
	d2Daemon, err := d2DaemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fa := NewMockConnectedAgents(ctrl)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	caErr := puller.getLeasesFromDaemon(caDaemon)
	d2Err := puller.getLeasesFromDaemon(d2Daemon)

	// Assert
	require.NoError(t, caErr)
	require.NoError(t, d2Err)
}

// Test getLeasesFromDaemon to ensure that it returns an error when
// ReceiveKeaLeases returns an error.
func TestGetLeasesFromDaemonWhenReceiveKeaLeasesReturnsErr(t *testing.T) {
	// Arrange
	errorText := "According to all known laws of aviation, there is no way a bee should be able to fly."
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeases := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{nil},
		[]error{errors.New(errorText)},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.getLeasesFromDaemon(daemon)

	// Assert
	require.ErrorContains(t, err, errorText)
}

// Test getLeasesFromDaemon to ensure that it returns an error when
// there's a nil in the stream of leases.
func TestGetLeasesFromDaemonWhenReceiveKeaLeasesReturnsNilInStream(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeases := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{
				Lease: &agentapi.Lease{
					IpAddress:     "192.168.1.180",
					HwAddress:     "00:01:02:03:04:05",
					Cltt:          1000,
					ValidLifetime: 3600,
					Family:        4,
					SubnetID:      10,
					State:         0,
				},
			},
			nil,
			{
				Lease: &agentapi.Lease{
					IpAddress:     "192.168.1.27",
					HwAddress:     "00:01:02:03:04:06",
					Cltt:          1009,
					ValidLifetime: 3600,
					Family:        4,
					SubnetID:      10,
					State:         0,
				},
			},
		},
		[]error{nil, nil, nil},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.getLeasesFromDaemon(daemon)

	// Assert
	require.ErrorContains(t, err, "unexpected nil")
}

// Test getLeasesFromDaemon to ensure that it returns an error when
// there's an invalid lease.
func TestGetLeasesFromDaemonWhenReceiveKeaLeasesReturnsInvalidLease(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeases := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{Lease: nil},
		},
		[]error{nil},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeases)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.getLeasesFromDaemon(daemon)

	// Assert
	require.ErrorContains(t, err, "unable to convert")
}

// Test getLeasesFromDaemon to ensure that it correctly reads the maximum CLTT
// out of the database when fetching from a daemon the second time.
func TestGetLeasesFromDaemonReadsMaxCLTTFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeasesFirst := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{
				Lease: &agentapi.Lease{
					IpAddress:     "192.168.1.180",
					HwAddress:     "00:01:02:03:04:05",
					Cltt:          1000,
					ValidLifetime: 3600,
					Family:        4,
					SubnetID:      10,
					State:         0,
				},
			},
		},
		[]error{nil},
	)
	mockLeasesSecond := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{
				Lease: &agentapi.Lease{
					IpAddress:     "192.168.1.27",
					HwAddress:     "00:01:02:03:04:06",
					Cltt:          1009,
					ValidLifetime: 3600,
					Family:        4,
					SubnetID:      10,
					State:         0,
				},
			},
		},
		[]error{nil},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).Return(mockLeasesFirst)
	fa.EXPECT().ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(1000))).Return(mockLeasesSecond)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	errFirst := puller.getLeasesFromDaemon(daemon)
	errSecond := puller.getLeasesFromDaemon(daemon)

	// Assert
	require.NoError(t, errFirst)
	require.NoError(t, errSecond)
}

// Test getLeasesFromDaemon to ensure that the error occurred when the lease
// is added to the database is handled property.
func TestGetLeasesFromDaemonAddLeaseError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.InitializeSettings(db, 0)

	daemonServer, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := daemonServer.Configure(`{ "Dhcp4": {
		"lease-database": {
        	"type": "memfile",
        	"lfc-interval": 3600,
        	"name": "/var/lib/kea/kea-leases4.csv"
    	}
	}}`)
	require.NoError(t, err)
	daemon, err := daemonServer.GetDaemon()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeases := storkutil.ZipPairs(
		[]*agentapi.ReceiveKeaLeasesRsp{
			{Lease: &agentapi.Lease{
				Family:        4,
				IpAddress:     "192.0.2.1",
				Cltt:          42,
				ValidLifetime: 420,
			}},
		},
		[]error{nil},
	)
	fa := NewMockConnectedAgents(ctrl)
	fa.EXPECT().
		ReceiveKeaLeases(gomock.Any(), gomock.Any(), gomock.Eq(uint64(0))).
		Do(func(ctx context.Context, daemon agentcomm.ControlledDaemon, minCLTT uint64) {
			// Tear down the database to cause an error when the puller tries
			// to add leases to it.
			teardown()
		}).
		Return(mockLeases)

	puller, err := NewLeasesPuller(db, fa)
	require.NoError(t, err)
	defer puller.Shutdown()

	// Act
	err = puller.getLeasesFromDaemon(daemon)

	// Assert
	require.ErrorContains(t, err, "problem inserting lease")
}
