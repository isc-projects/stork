package configreview

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Creates review context from configuration string.
func createReviewContext(t *testing.T, db *dbops.PgDB, configStr string) *ReviewContext {
	config, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)

	// Configuration must contain one of the keywords that identify the
	// daemon type.
	daemonName := dbmodel.DaemonNameDHCPv4
	if strings.Contains(configStr, "Dhcp6") {
		daemonName = dbmodel.DaemonNameDHCPv6
	}
	// Create the daemon instance and the context.
	ctx := newReviewContext(db, &dbmodel.Daemon{
		ID:   1,
		Name: daemonName,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	require.NotNil(t, ctx)

	return ctx
}

// Creates a new host with IP reservations in the database. Adding a host
// requires a machine, app and subnet which are also added by this function.
func createHostInDatabase(t *testing.T, db *dbops.PgDB, configStr, subnetPrefix string, reservationAddress ...string) {
	// Detect whether we're dealing with DHCPv4 or DHCPv6.
	daemonName := dbmodel.DaemonNameDHCPv4
	parsedPrefix := storkutil.ParseIP(subnetPrefix)
	if parsedPrefix != nil && parsedPrefix.Protocol == storkutil.IPv6 {
		daemonName = dbmodel.DaemonNameDHCPv6
	}
	// Create the machine.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	config, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)

	// Create the app.
	app := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   daemonName,
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: config,
				},
			},
		},
	}
	addedDaemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	// Create the subnet.
	subnet := dbmodel.Subnet{
		Prefix: subnetPrefix,
	}
	err = dbmodel.AddSubnet(db, &subnet)
	require.NoError(t, err)

	// Associate the daemon with the subnet.
	err = dbmodel.AddDaemonToSubnet(db, &subnet, app.Daemons[0])
	require.NoError(t, err)

	// Add the host for this subnet.
	host := &dbmodel.Host{
		SubnetID: subnet.ID,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
	}
	// Append reserved addresses.
	for _, a := range reservationAddress {
		host.IPReservations = append(host.IPReservations, dbmodel.IPReservation{
			Address: a,
		})
	}
	// Add the host.
	err = dbmodel.AddHost(db, host)
	require.NoError(t, err)

	// Associate the daemon with the host.
	err = dbmodel.AddDaemonToHost(db, host, app.Daemons[0].ID, "api", 1)
	require.NoError(t, err)
}

// Tests that the checker checking stat_cmds hooks library presence
// returns nil when the library is loaded.
func TestStatCmdsPresent(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_stat_cmds.so"
                }
            ]
        }
    }`
	report, err := statCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking stat_cmds hooks library presence
// returns the report when the library is not loaded.
func TestStatCmdsAbsent(t *testing.T) {
	configStr := `{"Dhcp4": { }}`
	report, err := statCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "The Kea Statistics Commands library")
}

// Tests that the checker checking host_cmds hooks library presence
// returns nil when the library is loaded.
func TestHostCmdsPresent(t *testing.T) {
	// The host backend is in use and the library is loaded.
	configStr := `{
        "Dhcp4": {
            "hosts-database": [
                {
                    "type": "mysql"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds presence takes into
// account whether or not the host-database(s) parameters are
// also specified.
func TestHostCmdsBackendUnused(t *testing.T) {
	// The backend is not used and the library is not loaded.
	// There should be no report.
	configStr := `{
        "Dhcp4": { }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// host-database (singular) parameter is specified.
func TestHostCmdsAbsentHostsDatabase(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	configStr := `{
        "Dhcp4": {
            "hosts-database": {
                "type": "mysql"
            }
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// hosts-databases (plural) parameter is specified.
func TestHostCmdsAbsentHostsDatabases(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	configStr := `{
        "Dhcp4": {
            "hosts-databases": [
                {
                    "type": "mysql"
                }
            ]
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}

// Tests that the checker finding dispensable shared networks finds
// an empty IPv4 shared network.
func TestSharedNetworkDispensableNoDHCPv4Subnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        },
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 1 empty shared network")
}

// Tests that the checker finding dispensable shared networks finds
// an IPv4 shared network with a single subnet.
func TestSharedNetworkDispensableSingleDHCPv4Subnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 1 shared network with only a single subnet")
}

// Tests that the checker finding dispensable shared networks finds
// multiple empty IPv4 shared networks and multiple Ipv4 shared networks
// with a single subnet.
func TestSharedNetworkDispensableSomeEmptySomeWithSingleSubnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar"
                },
                {
                    "name": "baz",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                },
                {
                    "name": "zab",
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                },
                {
                    "name": "bac",
                    "subnet4": [
                        {
                            "subnet": "192.0.4.0/24"
                        },
                        {
                            "subnet": "192.0.5.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 2 empty shared networks and 2 shared networks with only a single subnet")
}

// Tests that the checker finding dispensable shared networks does not
// generate a report when there are no empty shared networks nor the
// shared networks with a single subnet.
func TestSharedNetworkDispensableMultipleDHCPv4Subnets(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        },
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable shared networks finds
// an empty IPv6 shared network.
func TestSharedNetworkDispensableNoDHCPv6Subnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        },
                        {
                            "subnet": "2001:db8:2::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 1 empty shared network")
}

// Tests that the checker finding dispensable shared networks finds
// an IPv6 shared network with a single subnet.
func TestSharedNetworkDispensableSingleDHCPv6Subnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 1 shared network with only a single subnet")
}

// Tests that the checker finding dispensable shared networks does not
// generate a report when there are no empty shared networks nor the
// shared networks with a single subnet.
func TestSharedNetworkDispensableMultipleDHCPv6Subnets(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        },
                        {
                            "subnet": "2001:db8:2::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets finds the subnets
// that comprise no pools and no reservations.
func TestIPv4SubnetDispensableNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets finds the subnets
// that have no reservations in the database.
func TestIPv4SubnetDispensableNoPoolsNoReservationsHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "id": 111,
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "id": 222,
                    "subnet": "192.0.3.0/24"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets generates no report
// when there are host reservations for these subnets in the database.
func TestIPv4SubnetDispensableSomeDatabaseReservations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 111,
                    "subnet": "192.0.3.0/24"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create a host in the database.
	createHostInDatabase(t, db, configStr, "192.0.3.0/24", "192.0.3.50")

	report, err := subnetDispensable(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when pools are present.
func TestIPv4SubnetDispensableSomePoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when reservations are present.
func TestIPv4SubnetDispensableNoPoolsSomeReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations": [
                        {
                            "ip-address": "192.0.3.10",
                            "hw-address": "01:02:03:04:05:06"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets finds the subnets
// that comprise no pools and no reservations.
func TestIPv6SubnetDispensableNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ],
            "subnet6": [
                {
                    "subnet": "2001:db8:2::/64"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets finds the subnets
// that comprise no reservations in the host database.
func TestIPv6SubnetDispensableNoPoolsNoReservationsHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "id": 111,
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ],
            "subnet6": [
                {
                    "id": 222,
                    "subnet": "2001:db8:2::/64"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration includes 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets generates no report
// when there are host reservations for these subnets in the database.
func TestIPv6SubnetDispensableSomeDatabaseReservations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create a host in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "2001:db8:1::50", "3000::/96")

	report, err := subnetDispensable(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when pools are present.
func TestIPv6SubnetDispensableSomePoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::5 - 2001:db8:1::15"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when prefix delegation pools are present.
func TestIPv6SubnetDispensableSomePdPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3001::/16",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when reservations are present.
func TestIPv6SubnetDispensableNoPoolsSomeReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::10" ],
                            "hw-address": "01:02:03:06:05:06"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list.
func TestDHCPv4ReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks.
func TestDHCPv4ReservationsOutOfPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the global level.
func TestDHCPv4ReservationsOutOfPoolEnabledGlobally(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": true,
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the shared network level.
func TestDHCPv4ReservationsOutOfPoolEnabledAtSharedNetworkLevel(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": false,
            "shared-networks": [
                {
                    "reservation-mode": "out-of-pool",
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the subnet level.
func TestDHCPv4ReservationsOutOfPoolEnabledAtSubnetLevel(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": false,
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations-out-of-pool": true,
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when there are
// no reservations in the subnet.
func TestDHCPv4ReservationsOutOfPoolNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns the report when a subnet has
// reservations but no pools.
func TestDHCPv4ReservationsOutOfPoolNoPools(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// no reservations.
func TestDHCPv4ReservationsOutOfPoolNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24"
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// reservations but they contain no IP addresses.
func TestDHCPv4ReservationsOutOfPoolNoPoolsNonIPReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "hostname": "myhost123.example.org"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list. Hosts in the database case.
func TestDHCPv4DatabaseReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 111,
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "192.0.3.0/24", "192.0.3.5")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used ignores hosts specified in the
// database when host_cmds is unused.
func TestDHCPv4DatabaseReservationsOutOfPoolNoHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 111,
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "192.0.3.0/24", "192.0.3.5")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used ignores hosts lacking IP reservations.
func TestDHCPv4DatabaseReservationsOutOfPoolNoIPReservation(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 111,
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database without
	// any IP reservation.
	createHostInDatabase(t, db, configStr, "192.0.3.0/24")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list. Prefix delegation case.
func TestDHCPv6ReservationsOutOfPDPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ],
                    "reservations": [
                        {
                            "prefixes": [ "3001::/96" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when reserved
// IP address is within the pool.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnetInPool(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::30" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when reserved
// delegated prefix is within the prefix delegation pool.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnetInPDPool(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ],
                    "reservations": [
                        {
                            "prefixes": [ "3000::/96" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks.
func TestDHCPv6ReservationsOutOfPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks. Prefix delegation case.
func TestDHCPv6ReservationsOutOfPDPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pd-pools": [
                                {
                                    "prefix": "3000::",
                                    "prefix-len": 64,
                                    "delegated-len": 96
                                }
                            ],
                            "reservations": [
                                {
                                    "prefixes": [ "3001::/96" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the global level.
func TestDHCPv6ReservationsOutOfPoolEnabledGlobally(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": true,
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the shared network level.
func TestDHCPv6ReservationsOutOfPoolEnabledAtSharedNetworkLevel(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": false,
            "shared-networks": [
                {
                    "reservation-mode": "out-of-pool",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the subnet level.
func TestDHCPv6ReservationsOutOfPoolEnabledAtSubnetLevel(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": false,
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations-out-of-pool": true,
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when there are
// no reservations in the subnet.
func TestDHCPv6ReservationsOutOfPoolNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns the report when a subnet has
// reservations but no pools.
func TestDHCPv6ReservationsOutOfPoolNoPools(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// no reservations.
func TestDHCPv6ReservationsOutOfPoolNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64"
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// reservations but they contain neither IP addresses nor delegated
// prefixes.
func TestDHCPv6ReservationsOutOfPoolNoPoolsNonIPReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "hostname": "myhost123.example.org"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list. Hosts in the database case.
func TestDHCPv6DatabaseReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "2001:db8:1::5")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list. Hosts in the database and prefix delegation case.
func TestDHCPv6DatabaseReservationsOutOfPDPoolTopLevelSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "3001::/96")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when IP address
// reservation is in pool.
func TestDHCPv6DatabaseReservationsOutOfPoolTopLevelSubnetInPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "2001:db8:1::50")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when delegated
// prefix reservation is in pool.
func TestDHCPv6DatabaseReservationsOutOfPDPoolTopLevelSubnetInPool(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "3000::/96")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used ignores hosts specified in the
// database when host_cmds is unused.
func TestDHCPv6DatabaseReservationsOutOfPoolNoHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64", "2001:db8:1::5")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used ignores hosts lacking IP reservations.
func TestDHCPv6DatabaseReservationsOutOfPoolNoIPReservation(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 111,
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ]
        }
    }`

	// Create the out-of-pool host reservation in the database without
	// any IP reservation.
	createHostInDatabase(t, db, configStr, "2001:db8:1::/64")

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Benchmark measuring performance of a Kea configuration checker that detects
// subnets in which the out-of-pool host reservation mode is recommended.
func BenchmarkReservationsOutOfPoolConfig(b *testing.B) {
	// Create 10.000 subnets with a pool and out of pool reservation.
	subnets := []interface{}{}
	for i := 0; i < 10000; i++ {
		prefix := fmt.Sprintf("192.%d.%d", i/256, i%256)
		subnet := map[string]interface{}{
			"subnet": fmt.Sprintf("%s.0/24", prefix),
			"pools": []map[string]interface{}{
				{
					"pool": fmt.Sprintf("%s.10 - %s.100", prefix, prefix),
				},
			},
			"reservations": []map[string]interface{}{
				{
					"ip-address": fmt.Sprintf("%s.5", prefix),
				},
			},
		}
		subnets = append(subnets, subnet)
	}

	// Create Kea DHCPv4 configuration with the subnets.
	configMap := map[string]interface{}{
		"Dhcp4": map[string]interface{}{
			"subnet4": subnets,
		},
	}
	configStr, err := json.Marshal(configMap)
	if err != nil {
		b.Fatalf("failed to marshal configuration map: %+v", err)
	}
	config, err := dbmodel.NewKeaConfigFromJSON(string(configStr))
	if err != nil {
		b.Fatalf("failed to create new Kea configuration from JSON: %+v", err)
	}

	// The benchmark starts here.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := newReviewContext(nil, &dbmodel.Daemon{
			ID:   1,
			Name: dbmodel.DaemonNameDHCPv4,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: config,
			},
		}, false, nil)
		_, err = reservationsOutOfPool(ctx)
		if err != nil {
			b.Fatalf("checker failed: %+v", err)
		}
	}
}

// Benchmark measuring performance of a Kea configuration checker that detects
// subnets in which the out-of-pool host reservation mode is recommended.
// This benchmark stores host reservations in the database.
func BenchmarkReservationsOutOfPoolDatabase(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	// Create the machine.
	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	if err != nil {
		b.Fatalf("failed to add a machine: %+v", err)
	}

	// Create the app.
	app := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   dbmodel.DaemonNameDHCPv4,
				Active: true,
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	if err != nil {
		b.Fatalf("failed to add an app: %+v", err)
	}

	// Create 10.000 subnets with a pool and out of pool reservation.
	subnets := []interface{}{}
	for i := 0; i < 10000; i++ {
		prefix := fmt.Sprintf("192.%d.%d", i/256, i%256)
		subnet := map[string]interface{}{
			"subnet": fmt.Sprintf("%s.0/24", prefix),
			"pools": []map[string]interface{}{
				{
					"pool": fmt.Sprintf("%s.10 - %s.100", prefix, prefix),
				},
			},
			"hooks-libraries": []map[string]interface{}{
				{
					"library": "/usr/lib/kea/libdhcp_host_cmds.so",
				},
			},
		}
		subnets = append(subnets, subnet)

		// Create the subnet in the database.
		dbSubnet := dbmodel.Subnet{
			Prefix: prefix,
		}
		err = dbmodel.AddSubnet(db, &dbSubnet)
		if err != nil {
			b.Fatalf("failed to add a subnet %s: %+v", dbSubnet.Prefix, err)
		}
		// Associate the daemon with the subnet.
		err = dbmodel.AddDaemonToSubnet(db, &dbSubnet, app.Daemons[0])
		if err != nil {
			b.Fatalf("failed to add app to subnet %s: %+v", dbSubnet.Prefix, err)
		}
		// Add the host for this subnet.
		host := &dbmodel.Host{
			SubnetID: dbSubnet.ID,
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: fmt.Sprintf("%s.5", prefix),
				},
			},
		}
		// Add the host.
		err = dbmodel.AddHost(db, host)
		if err != nil {
			b.Fatalf("failed to add app to subnet %s: %+v", dbSubnet.Prefix, err)
		}
		// Associate the daemon with the host.
		err = dbmodel.AddDaemonToHost(db, host, app.Daemons[0].ID, "api", 1)
		if err != nil {
			b.Fatalf("failed to add app to host: %+v", err)
		}
	}

	// Create Kea DHCPv4 configuration with the subnets.
	configMap := map[string]interface{}{
		"Dhcp4": map[string]interface{}{
			"subnet4": subnets,
		},
	}
	configStr, err := json.Marshal(configMap)
	if err != nil {
		b.Fatalf("failed to marshal configuration map: %+v", err)
	}
	config, err := dbmodel.NewKeaConfigFromJSON(string(configStr))
	if err != nil {
		b.Fatalf("failed to create new Kea configuration from JSON: %+v", err)
	}

	// The benchmark starts here.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := newReviewContext(db, &dbmodel.Daemon{
			ID:   1,
			Name: dbmodel.DaemonNameDHCPv4,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: config,
			},
		}, false, nil)
		_, err = reservationsOutOfPool(ctx)
		if err != nil {
			b.Fatalf("checker failed: %+v", err)
		}
	}
}
