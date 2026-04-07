package kea

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

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
func daemonWithHAHotStandbyTriangle(d *dbmodel.Daemon, standbyFor, primaryFor int64) *dbmodel.Daemon {
	myID := d.ID
	hooks := keaconfig.HookLibraries{
		{
			Library: "/usr/lib/kea/libdhcp_ha.so",
			Parameters: (json.RawMessage)(fmt.Sprintf(`{
				"high-availability": [
					{
						"this-server-name": "server%d",
						"mode": "hot-standby",
						"heartbeat-delay": 10000,
						"peers": [{
							"name": "server%d",
							"url": "http://192.168.1.%d:8005/",
							"role": "primary",
							"auto-failover": true
						}, {
							"name": "server%d",
							"url": "http://192.168.1.%d:8005/",
							"role": "standby",
							"auto-failover": true
						}]
					},
					{
						"this-server-name": "server%d",
						"mode": "hot-standby",
						"heartbeat-delay": 10000,
						"peers": [{
							"name": "server%d",
							"url": "http://192.168.1.%d:8005/",
							"role": "primary",
							"auto-failover": true
						}, {
							"name": "server%d",
							"url": "http://192.168.1.%d:8005/",
							"role": "standby",
							"auto-failover": true
						}]
					}
				]
			}`,
				myID,       // this-server-name
				myID,       // primary name
				myID,       // primary url
				primaryFor, // secondary name
				primaryFor, // secondary url
				myID,       // this-server-name
				standbyFor, // primary name
				standbyFor, // primary url
				myID,       // secondary name
				myID,       // secondary url
			)),
		},
	}
	switch d.Name {
	case daemonname.DHCPv4:
		d.KeaDaemon.Config.DHCPv4Config.HookLibraries = hooks
	case daemonname.DHCPv6:
		d.KeaDaemon.Config.DHCPv6Config.HookLibraries = hooks
	default:
		panic("test function used incorrectly, must pass a DHCPv4 or DHCPv6 daemon.")
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
			daemonWithLeaseMemfile(
				mockFilterableDaemon(daemonname.DHCPv6, 20, 7),
				configuredLeasefilePath,
				nil,
			),
			22,
			21,
		)
		haDaemon15OnMachine8 = daemonWithHAHotStandbyTriangle(
			daemonWithLeaseMemfile(
				mockFilterableDaemon(daemonname.DHCPv6, 21, 8),
				configuredLeasefilePath,
				nil,
			),
			20,
			22,
		)
		haDaemon16OnMachine9 = daemonWithHAHotStandbyTriangle(
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
}
