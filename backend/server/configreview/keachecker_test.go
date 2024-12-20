package configreview

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Creates review context from configuration string.
func createReviewContext(t *testing.T, db *dbops.PgDB, configStr string, keaVersion string) *ReviewContext {
	config, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)

	// Configuration must contain one of the keywords that identify the
	// daemon type.
	daemonName := dbmodel.DaemonNameDHCPv4
	if strings.Contains(configStr, "Dhcp6") {
		daemonName = dbmodel.DaemonNameDHCPv6
	} else if strings.Contains(configStr, "Control-agent") {
		daemonName = dbmodel.DaemonNameCA
	}

	// Create the daemon instance and the context.
	ctx := newReviewContext(db, &dbmodel.Daemon{
		ID:      1,
		Name:    daemonName,
		Version: keaVersion,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
		// The subject daemon may lack the app. The dispatcher accepts the
		// daemon object and doesn't validate if it contains non-nil references
		// to the app and machine.
	}, []Trigger{ManualRun}, nil)
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

	localHost := &dbmodel.LocalHost{
		DaemonID:   app.Daemons[0].ID,
		DataSource: dbmodel.HostDataSourceAPI,
	}

	// Append reserved addresses.
	for _, a := range reservationAddress {
		localHost.IPReservations = append(localHost.IPReservations, dbmodel.IPReservation{
			Address: a,
		})
	}

	host.LocalHosts = append(host.LocalHosts, *localHost)

	// Add the host.
	err = dbmodel.AddHost(db, host)
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
	report, err := statCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking stat_cmds hooks library presence
// returns the report when the library is not loaded.
func TestStatCmdsAbsent(t *testing.T) {
	configStr := `{"Dhcp4": { }}`
	report, err := statCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "The Kea Statistics Commands library")
}

// Tests that the checker checking lease_cmds hooks library presence returns
// nil when the library is loaded.
func TestLeaseCmdsPresent(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_lease_cmds.so"
                }
            ]
        }
    }`
	report, err := leaseCmdsPresence(createReviewContext(
		t, nil, configStr, "2.2.0",
	))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking lease_cmds hooks library presence returns
// the report when the library is not loaded.
func TestLeaseCmdsAbsent(t *testing.T) {
	configStr := `{"Dhcp4": { }}`
	report, err := leaseCmdsPresence(createReviewContext(
		t, nil, configStr, "2.2.0",
	))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "The Kea Lease Commands library")
}

// Tests that the checker checking host_cmds hooks library presence
// returns nil when the library is loaded.
func TestHostCmdsPresent(t *testing.T) {
	// The host backend is in use and the library is loaded.
	configStr := `{
        "Dhcp4": {
            "hosts-databases": [
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
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea can be configured")
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
	report, err := hostCmdsPresence(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea can be configured")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 1 empty shared network")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 1 shared network with only a single subnet")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 2 empty shared networks and 2 shared networks with only a single subnet")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 1 empty shared network")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 1 shared network with only a single subnet")
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
	report, err := sharedNetworkDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 2 subnets without pools and host reservations")
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
	report, err := subnetDispensable(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 2 subnets without pools and host reservations")
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

	report, err := subnetDispensable(createReviewContext(t, db, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 2 subnets without pools and host reservations")
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
	report, err := subnetDispensable(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "configuration includes 2 subnets without pools and host reservations")
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

	report, err := subnetDispensable(createReviewContext(t, db, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
                            "prefix": "3001::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := subnetDispensable(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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
	report, err := reservationsOutOfPool(createReviewContext(t, nil, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "includes 1 subnet for which it is recommended to use out-of-pool")
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
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

	report, err := reservationsOutOfPool(createReviewContext(t, db, configStr, "2.2.0"))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that no overlaps are detected for empty subnet list.
func TestFindOverlapsEmptySubnets(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Empty(t, overlaps)
}

// Test that no overlaps are detected for non-overlapping subnets.
func TestFindOverlapsNonOverlappingSubnets(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.1.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "192.168.2.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     4,
				Subnet: "192.168.3.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:1::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     7,
				Subnet: "3001:2::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     8,
				Subnet: "3001:3::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Empty(t, overlaps)
}

// Test that the checker doesn't panic if a zero subnet occurs.
func TestFindOverlapsZeroSubnet(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "0.0.0.0/0",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "0.0.0.0/32",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "192.168.2.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     4,
				Subnet: "192.168.2.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "::/0",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "::/128",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     7,
				Subnet: "3001:2::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     8,
				Subnet: "3001:2::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Len(t, overlaps, 2)
}

// Test that duplicated prefixes are detected as overlaps.
func TestFindOverlapsForDuplicates(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:0::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Len(t, overlaps, 2)
	require.EqualValues(t, 1, overlaps[1].parent.GetID())
	require.EqualValues(t, 2, overlaps[1].child.GetID())
	require.EqualValues(t, 5, overlaps[0].parent.GetID())
	require.EqualValues(t, 6, overlaps[0].child.GetID())
}

// Test that duplicated prefixes are detected as overlaps even if the prefix is
// repeatedly duplicated.
func TestFindOverlapsForMultipleDuplicates(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:0::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     7,
				Subnet: "3001:0::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Len(t, overlaps, 6)
	require.EqualValues(t, 5, overlaps[0].parent.GetID())
	require.EqualValues(t, 6, overlaps[0].child.GetID())
	require.EqualValues(t, 5, overlaps[1].parent.GetID())
	require.EqualValues(t, 7, overlaps[1].child.GetID())
	require.EqualValues(t, 6, overlaps[2].parent.GetID())
	require.EqualValues(t, 7, overlaps[2].child.GetID())
	require.EqualValues(t, 1, overlaps[3].parent.GetID())
	require.EqualValues(t, 2, overlaps[3].child.GetID())
	require.EqualValues(t, 1, overlaps[4].parent.GetID())
	require.EqualValues(t, 3, overlaps[4].child.GetID())
	require.EqualValues(t, 2, overlaps[5].parent.GetID())
	require.EqualValues(t, 3, overlaps[5].child.GetID())
}

// Test that overlaps are detected for the same network but different prefix
// lengths.
func TestFindOverlapsForSameNetworkButDifferentPrefixLengths(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/16",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.0.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/64",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:0::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Len(t, overlaps, 2)
	require.EqualValues(t, 1, overlaps[1].parent.GetID())
	require.EqualValues(t, 2, overlaps[1].child.GetID())
	require.EqualValues(t, 5, overlaps[0].parent.GetID())
	require.EqualValues(t, 6, overlaps[0].child.GetID())
}

// Test that overlaps are detected when one prefix is contained by another.
func TestFindOverlapsForContainingPrefixes(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/16",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.5.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:0::/80",
			},
		},
	}
	// Act
	overlaps := findOverlaps(subnets, 42)

	// Assert
	require.Len(t, overlaps, 2)
	require.EqualValues(t, 1, overlaps[1].parent.GetID())
	require.EqualValues(t, 2, overlaps[1].child.GetID())
	require.EqualValues(t, 5, overlaps[0].parent.GetID())
	require.EqualValues(t, 6, overlaps[0].child.GetID())
}

// Test that the searching for overlaps is stopped if the limit is exceeded on
// duplicated subnets.
func TestFindOverlapsExceedLimitOnDuplicatedSubnets(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/16",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.5.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "192.68.5.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     4,
				Subnet: "192.68.5.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001:0::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:1::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     7,
				Subnet: "2001:0::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     8,
				Subnet: "2001:0::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     9,
				Subnet: "4001:0::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     10,
				Subnet: "4001:0::/16",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 2)

	// Assert
	require.Len(t, overlaps, 2)
	require.EqualValues(t, 7, overlaps[0].parent.GetID())
	require.EqualValues(t, 8, overlaps[0].child.GetID())
	require.EqualValues(t, 5, overlaps[1].parent.GetID())
	require.EqualValues(t, 6, overlaps[1].child.GetID())
}

// Test that the searching for overlaps is stopped if the limit of overlapping
// subnets is exceeded.
func TestFindOverlapsExceedLimitOnContainingSubnets(t *testing.T) {
	// Arrange
	subnets := []keaconfig.Subnet{
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     1,
				Subnet: "192.168.0.0/16",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     2,
				Subnet: "192.168.5.0/24",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     3,
				Subnet: "192.68.0.0/16",
			},
		},
		&keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     4,
				Subnet: "192.68.5.0/24",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     5,
				Subnet: "3001::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     6,
				Subnet: "3001:1::/80",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     7,
				Subnet: "2001::/16",
			},
		},
		&keaconfig.Subnet6{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     8,
				Subnet: "2001:1::/80",
			},
		},
	}

	// Act
	overlaps := findOverlaps(subnets, 2)

	// Assert
	require.Len(t, overlaps, 2)
	require.EqualValues(t, 7, overlaps[0].parent.GetID())
	require.EqualValues(t, 8, overlaps[0].child.GetID())
	require.EqualValues(t, 5, overlaps[1].parent.GetID())
	require.EqualValues(t, 6, overlaps[1].child.GetID())
}

// Test that error is generated for non-DHCP daemon.
func TestSubnetsOverlappingReportErrorForNonDHCPDaemon(t *testing.T) {
	// Arrange
	ctx := newReviewContext(nil, dbmodel.NewBind9Daemon(true), Triggers{ManualRun},
		func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that report is nil for non-overlapping subnets.
func TestSubnetsOverlappingReportForNonOverlappingSubnets(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": []
        }
    }`)
	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that report has a proper content for a single overlap.
func TestSubnetsOverlappingReportForSingleOverlap(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 1,
                    "subnet": "10.0.1.0/24"
                },
                {
                    "id": 2,
                    "subnet": "10.0.0.0/16"
                }
            ]
        }
    }`)
	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 42, report.daemonID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea {daemon} configuration includes 1 overlapping subnet pair.")
	require.Contains(t, *report.content, "1. [2] 10.0.0.0/16 is overlapped by [1] 10.0.1.0/24")
}

// Test that report has a proper content for a single overlap and subnets without IDs.
func TestSubnetsOverlappingReportForSingleOverlapAndNoSubnetIDs(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "10.0.1.0/24"
                },
                {
                    "subnet": "10.0.0.0/16"
                }
            ]
        }
    }`)
	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 42, report.daemonID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea {daemon} configuration includes 1 overlapping subnet pair.")
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "1. 10.0.0.0/16 is overlapped by 10.0.1.0/24")
}

// Test that report has a proper content for a multiple overlaps.
func TestSubnetsOverlappingReportForMultipleOverlap(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42

	var subnetsConfig []interface{}
	for i := 0; i < 12; i++ {
		subnetsConfig = append(subnetsConfig, map[string]interface{}{
			"id":     i + 1,
			"subnet": fmt.Sprintf("10.0.0.0/%d", 8+i),
		})
	}
	config, _ := json.Marshal(map[string]interface{}{
		"Dhcp4": map[string]interface{}{
			"subnet4": subnetsConfig,
		},
	})
	_ = daemon.SetConfigFromJSON(string(config))

	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 42, report.daemonID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea {daemon} configuration includes at least 10 overlapping subnet pairs.")
	require.Contains(t, *report.content, "1. [1] 10.0.0.0/8 is overlapped by [2] 10.0.0.0/9")
	require.Contains(t, *report.content, "10. [1] 10.0.0.0/8 is overlapped by [11] 10.0.0.0/18")
	require.NotContains(t, *report.content, "11.")
}

// Test that no error or overlaps are returned for a Kea config without subnet
// node.
func TestSubnetsOverlappingForMissingSubnetNode(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": { }
    }`)
	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that shared networks are processed by the overlapping checker.
func TestSubnetsOverlappingForSharedNetworks(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "10.0.1.0/24"
                        },
                        {
                            "subnet": "10.0.0.0/16"
                        }
                    ]
                }
            ]
        }
    }`)

	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := subnetsOverlapping(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 42, report.daemonID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea {daemon} configuration includes 1 overlapping subnet pair.")
	require.Contains(t, *report.content, "1. 10.0.0.0/16 is overlapped by 10.0.1.0/24")
}

// Test that the canonical prefix is recognized correctly.
func TestGetCanonicalPrefixForValidPrefixes(t *testing.T) {
	// Arrange
	prefixes := []string{
		"10.10.0.0/16",
		"192.168.1.0/24",
		"172.100.50.40/29",
		"127.0.0.1/32",
		"3001::/80",
	}

	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			// Act
			canonicalPrefix, result := getCanonicalPrefix(prefix)

			// Assert
			require.True(t, result)
			require.EqualValues(t, prefix, canonicalPrefix)
		})
	}
}

// Test that the prefix with many zeros is reduced to the canonical form.
func TestGetCanonicalPrefixShortestIPv6Form(t *testing.T) {
	// Arrange
	prefix := "2001:0000:0000:0000:0000::/64"

	// Act
	canonicalPrefix, result := getCanonicalPrefix(prefix)

	// Assert
	require.True(t, result)
	require.EqualValues(t, "2001::/64", canonicalPrefix)
}

// Test that the non-canonical prefix is recognized correctly.
func TestIsCanonicalPrefixForInvalidPrefixes(t *testing.T) {
	// Arrange
	data := [][]string{
		{"10.10.42.0/16", "10.10.0.0/16"},
		{"192.168.1.42/24", "192.168.1.0/24"},
		{"172.100.50.42/29", "172.100.50.40/29"},
		{"3001::42:0/80", "3001::/80"},
		{"2001:0000:0000:0000:0000::42/64", "2001::/64"},
	}

	for _, entry := range data {
		prefix := entry[0]
		expected := entry[1]

		t.Run(prefix, func(t *testing.T) {
			// Act
			validPrefix, result := getCanonicalPrefix(prefix)

			// Assert
			require.False(t, result)
			require.EqualValues(t, expected, validPrefix)
		})
	}
}

// Test that the canonical prefixes checker generates an expected report.
func TestCanonicalPrefixes(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 1,
                    "subnet": "192.168.0.0/16"
                },
                {
                    "id": 2,
                    "subnet": "192.168.1.2/24"
                }
            ],
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "10.0.0.0/8"
                        },
                        {
                            "subnet": "10.1.2.3/24"
                        },
                        {
                            "subnet": "10.1.2.3/16"
                        },
                        {
                            "subnet": "foobar"
                        }
                    ]
                }
            ]
        }
    }`)

	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := canonicalPrefixes(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 42, report.daemonID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Kea {daemon} configuration contains 4 non-canonical prefixes.")
	require.Contains(t, *report.content, "1. [2] 192.168.1.2/24 is invalid prefix, expected: 192.168.1.0/24;")
	require.Contains(t, *report.content, "4. foobar is invalid prefix")
}

// Test that the canonical prefixes report is not generated if all prefixes are valid.
func TestCanonicalPrefixesForValidPrefixes(t *testing.T) {
	// Arrange
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	daemon.ID = 42
	_ = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 1,
                    "subnet": "192.168.0.0/16"
                }
            ],
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "10.0.0.0/8"
                        }
                    ]
                }
            ]
        }
    }`)

	ctx := newReviewContext(nil, daemon,
		Triggers{ManualRun}, func(i int64, err error) {})

	// Act
	report, err := canonicalPrefixes(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the canonical prefixes report is not generated for an empty config.
func TestCanonicalPrefixesForEmptyConfig(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp4": { }
    }`, "2.2.0")

	// Act
	report, err := canonicalPrefixes(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the HA MT mode checker produces no report if the top
// multi-threading is disabled.
func TestHighAvailabilityMultiThreadingModeCheckerTopMultiThreadingDisabled(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": false
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "peers": [
                            {
                                "name": "foo",
                                "url": "http://foobar:8000"
                            },
                            {
                                "name": "bar",
                                "url": "http://barfoo:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA MT mode checker produces no report if the top
// multi-threading is disabled and the HA-level multi-threading is enabled.
func TestHighAvailabilityMultiThreadingModeCheckerTopMTDisabledHAMTEnabled(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": { 
            "enable-multi-threading": false
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true
                        },
                        "peers": [
                            {
                                "name": "foo",
                                "url": "http://foobar:8000"
                            },
                            {
                                "name": "bar",
                                "url": "http://barfoo:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA MT mode checker produces no report if the HA is not configured.
func TestHighAvailabilityMultiThreadingModeCheckerNoHAConfigured(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        }
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA MT mode checker produces a report if the top
// multi-threading is enabled but the HA is configured to use single thread.
func TestHighAvailabilityMultiThreadingModeCheckerSingleThreaded(t *testing.T) {
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "peers": [
                            {
                                "name": "foo",
                                "url": "http://foobar:8000"
                            },
                            {
                                "name": "bar",
                                "url": "http://barfoo:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.NotNil(t, report)
	require.NoError(t, err)

	require.Len(t, report.refDaemonIDs, 1)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.refDaemonIDs[0])
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "daemon is configured to work "+
		"in multi-threading mode, but the High Availability hooks use "+
		"single-thread mode")
}

// Test that single threaded HA configuration in one of the relationships is
// properly detected in Kea 2.4.0.
func TestHighAvailabilityMultiThreadingModeCheckerSingleThreadedMultipleRelationships(t *testing.T) {
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
		"hooks-libraries": [
			{
				"library": "/libdhcp_ha.so",
				"parameters": {
					"high-availability": [
						{
							"peers": [
								{
									"name": "foo",
									"url": "http://foobar:8000"
								},
								{
									"name": "bar",
									"url": "http://barfoo:8000"
								}
							]
						},
						{
							"multi-threading": {
								"enable-multi-threading": false
							},
							"peers": [
								{
									"name": "baz",
									"url": "http://foobar:8000"
								},
								{
									"name": "zab",
									"url": "http://barfoo:8000"
								}
							]
						}
					]
				}
			}
		]
    } }`, "2.4.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.NotNil(t, report)
	require.NoError(t, err)

	require.Len(t, report.refDaemonIDs, 1)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.refDaemonIDs[0])
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "daemon is configured to work "+
		"in multi-threading mode, but the High Availability hooks use "+
		"single-thread mode")
}

// Test that the HA MT mode checker produces no report if the configuration
// contains no issues.
func TestHighAvailabilityMultiThreadingModeCheckerCorrectConfiguration(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true
                        },
                        "peers": [
                            {
                                "role": "primary",
                                "name": "foo",
                                "url": "http://foobar:8001"
                            },
                            {
                                "role": "standby",
                                "name": "bar",
                                "url": "http://barfoo:8001"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityMultiThreadingMode(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that no issue is returned when MT is enabled at the global level
// and for all HA relationships in Kea 2.4.0.
func TestHighAvailabilityMultiThreadingModeCheckerCorrectConfigurationMultipleRelationships(t *testing.T) {
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
		"multi-threading": {
			"enable-multi-threading": true
		},
		"hooks-libraries": [
			{
				"library": "/libdhcp_ha.so",
				"parameters": {
					"high-availability": [
						{
							"peers": [
								{
									"name": "foo",
									"url": "http://foobar:8000"
								},
								{
									"name": "bar",
									"url": "http://barfoo:8000"
								}
							]
						},
						{
							"peers": [
								{
									"name": "baz",
									"url": "http://foobar:8000"
								},
								{
									"name": "zab",
									"url": "http://barfoo:8000"
								}
							]
						}
					]
				}
			}
		]
    } }`, "2.4.0")

	report, err := highAvailabilityMultiThreadingMode(ctx)

	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA dedicated ports checker produces no report if the global
// multi-threading is not configured.
func TestHighAvailabilityDedicatedPortsCheckerNoGlobalMultiThreading(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": { } }`, "2.2.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA dedicated ports checker produces no report if the HA
// configuration is missing.
func TestHighAvailabilityDedicatedPortsCheckerMissingHAHook(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [ ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA dedicated ports checker produces no report if the HA
// doesn't use the multi-threading.
func TestHighAvailabilityDedicatedPortsCheckerMissingMultiThreading(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": false
                        }
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA dedicated ports checker produces a report if any peer uses
// the port assigned to the CA daemon.
func TestHighAvailabilityDedicatedPortsCheckerPortCollisionWithCA(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initialize the failover entries.
	failoverMachine := &dbmodel.Machine{
		Address:   "10.0.0.2",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, failoverMachine)

	failoverApp := &dbmodel.App{
		MachineID: failoverMachine.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "127.0.0.1",
				Port:    8000,
			},
		},
		Daemons: []*dbmodel.Daemon{{Name: dbmodel.DaemonNameCA}},
	}
	_, _ = dbmodel.AddApp(db, failoverApp)

	// Prepare the subject entries.
	ctx := createReviewContext(t, db, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true,
                            "http-dedicated-listener": true
                        },
                        "peers": [
                            {
                                "role": "primary",
                                "name": "bar",
                                "url": "http://10.0.0.2:8000"
                            },
                            {
                                "role": "standby",
                                "name": "baz",
                                "url": "http://10.0.0.3:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// The default IDs are already stored in the database.
	ctx.subjectDaemon.ID = 2
	ctx.subjectDaemon.AppID = 2
	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.NotNil(t, report)
	require.NoError(t, err)

	require.Len(t, report.refDaemonIDs, 2)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.Contains(t, report.refDaemonIDs, failoverApp.Daemons[0].ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"High Availability hook configured to use dedicated HTTP "+
			"listeners but the connections to the HA 'bar' peer with "+
			"the 'http://10.0.0.2:8000' URL are performed over the Kea Control Agent "+
			"omitting the dedicated HTTP listener of this peer. ")
}

// Test that the HA dedicated ports checker produces a report if the dedicated
// HTTP listener is not enabled.
func TestHighAvailabilityDedicatedPortsCheckerDedicatedListenerDisabled(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true,
                            "http-dedicated-listener": false
                        },
                        "peers": [
                            {
                                "role": "primary",
                                "name": "bar",
                                "url": "http://10.0.0.2:8000"
                            },
                            {
                                "role": "standby",
                                "name": "baz",
                                "url": "http://10.0.0.3:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.NotNil(t, report)
	require.NoError(t, err)

	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"is not configured to use dedicated HTTP listeners")
}

// Test that the HA dedicated ports checker produces a report if the dedicated
// HTTP listener is not enabled for any of the relationships.

func TestHighAvailabilityDedicatedPortsCheckerDedicatedListenerDisabledMultipleRelationships(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": {
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [
                        {
                            "multi-threading": {
                                "enable-multi-threading": true,
                                "http-dedicated-listener": true
                            },
                            "peers": [
                                {
                                    "role": "primary",
                                    "name": "bar",
                                    "url": "http://10.0.0.2:8000"
                                },
                                {
                                    "role": "standby",
                                    "name": "baz",
                                    "url": "http://10.0.0.3:8000"
                                }
                            ]
                        },
                        {
                            "multi-threading": {
                                "http-dedicated-listener": false
                            },
                            "peers": [
                                {
                                    "role": "primary",
                                    "name": "bar",
                                    "url": "http://10.0.0.2:8000"
                                },
                                {
                                    "role": "standby",
                                    "name": "baz",
                                    "url": "http://10.0.0.3:8000"
                                }
                            ]
                        }
                    ]
                }
            }
        ]
    } }`, "2.4.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.NotNil(t, report)
	require.NoError(t, err)

	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"is not configured to use dedicated HTTP listeners")
}

// Test that the HA dedicated ports checker produces no report if the
// configuration contains no issue.
func TestHighAvailabilityDedicatedPortsCheckerCorrectConfiguration(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initialize the failover entries.
	failoverMachine := &dbmodel.Machine{
		Address:   "10.0.0.2",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, failoverMachine)

	failoverApp := &dbmodel.App{
		MachineID: failoverMachine.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "10.0.0.2",
				Port:    8000,
			},
		},
		Daemons: []*dbmodel.Daemon{{Name: dbmodel.DaemonNameCA}},
	}
	_, _ = dbmodel.AddApp(db, failoverApp)

	// Prepare the subject entries.
	ctx := createReviewContext(t, db, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true,
                            "http-dedicated-listener": true
                        },
                        "peers": [
                            {
                                "role": "primary",
                                "name": "bar",
                                "url": "http://10.0.0.2:8001"
                            },
                            {
                                "role": "standby",
                                "name": "baz",
                                "url": "http://10.0.0.3:8001"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// The default IDs are already stored in the database.
	ctx.subjectDaemon.ID = 2
	ctx.subjectDaemon.AppID = 2

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the HA dedicated ports checker produces no report if the
// configuration contains no issue for multiple relationships.
func TestHighAvailabilityDedicatedPortsCheckerCorrectConfigurationMultipleRelationships(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initialize the failover entries.
	failoverMachine := &dbmodel.Machine{
		Address:   "10.0.0.2",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, failoverMachine)

	failoverApp := &dbmodel.App{
		MachineID: failoverMachine.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "10.0.0.2",
				Port:    8000,
			},
		},
		Daemons: []*dbmodel.Daemon{{Name: dbmodel.DaemonNameCA}},
	}
	_, _ = dbmodel.AddApp(db, failoverApp)

	// Prepare the subject entries.
	ctx := createReviewContext(t, db, `{ "Dhcp4": {
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [
                        {
                            "peers": [
                                {
                                    "role": "primary",
                                    "name": "bar",
                                    "url": "http://10.0.0.2:8001"
                                },
                                {
                                    "role": "standby",
                                    "name": "baz",
                                    "url": "http://10.0.0.3:8001"
                                }
                            ]
                        },
                        {
                            "multi-threading": {
                                "enable-multi-threading": true,
                                "http-dedicated-listener": true
                            },
                            "peers": [
                                {
                                    "role": "primary",
                                    "name": "baz",
                                    "url": "http://10.0.0.2:8001"
                                },
                                {
                                    "role": "standby",
                                    "name": "zab",
                                    "url": "http://10.0.0.3:8001"
                                }
                            ]
                        }

                    ]
                }
            }
        ]
    } }`, "2.4.0")

	// The default IDs are already stored in the database.
	ctx.subjectDaemon.ID = 2
	ctx.subjectDaemon.AppID = 2

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the port collision is detected if it occurs on the machine of the
// subject daemon.
func TestHighAvailabilityDedicatedPortsCheckerLocalPeer(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initialize the failover entries.
	machine := &dbmodel.Machine{
		Address:   "10.0.0.1",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, machine)

	failoverApp := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "127.0.0.1",
				Port:    8000,
			},
		},
		Daemons: []*dbmodel.Daemon{{Name: dbmodel.DaemonNameCA}},
	}
	_, _ = dbmodel.AddApp(db, failoverApp)

	// Prepare the subject entries.
	ctx := createReviewContext(t, db, `{ "Dhcp4": {
        "multi-threading": {
            "enable-multi-threading": true
        },
        "hooks-libraries": [
            {
                "library": "/libdhcp_ha.so",
                "parameters": {
                    "high-availability": [{
                        "multi-threading": {
                            "enable-multi-threading": true,
                            "http-dedicated-listener": true
                        },
                        "peers": [
                            {
                                "role": "primary",
                                "name": "bar",
                                "url": "http://10.0.0.2:8000"
                            },
                            {
                                "role": "standby",
                                "name": "baz",
                                "url": "http://10.0.0.1:8000"
                            }
                        ]
                    }]
                }
            }
        ]
    } }`, "2.2.0")

	// Act
	report, err := highAvailabilityDedicatedPorts(ctx)

	// Assert
	require.NoError(t, err)

	require.NotNil(t, report)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"High Availability hook configured to use dedicated HTTP "+
			"listeners but the connections to the HA 'baz' peer with "+
			"the 'http://10.0.0.1:8000' URL are performed over the Kea Control Agent "+
			"omitting the dedicated HTTP listener of this peer. ")
}

// Test that the error is returned if the non-DHCP daemon is checking.
func TestAddressPoolsExhaustedByReservationsForNonDHCPDaemonConfig(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Control-agent": { }
    }`, "2.2.0")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.Nil(t, report)
	require.ErrorContains(t, err, "unsupported daemon")
}

// Test that the no error and no issue are returned if the configuration
// doesn't contain subnets.
func TestAddressPoolsExhaustedByReservationsForMissingSubnets(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp4": {}
    }`, "2.2.0")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.Nil(t, report)
	require.Nil(t, err)
}

// Test that the no issue report is returned if the number of reservations is
// less then the number of available addresses in pool.
func TestAddressPoolsExhaustedByReservationsForLessReservationsThanAddresses(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "subnet": "fe80::/16",
                "pools": [
                    {
                        "pool": "fe80::1-fe80::3"
                    }
                ],
                "reservations": [
                    {
                        "ip-addresses": [
                            "fe80::1",
                            "fe80::2"
                        ]
                    }
                ]
            }]
        }
    }`, "2.2.0")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the issue report is returned if all pool addresses are reserved.
func TestAddressPoolsExhaustedByReservationsForEqualReservationsAndAddresses(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "subnet": "fe80::/16",
                "pools": [
                    {
                        "pool": "fe80::1-fe80::3"
                    }
                ],
                "reservations": [
                    {
                        "ip-addresses": [
                            "fe80::1",
                            "fe80::2"
                        ]
                    },
                    {
                        "ip-addresses": [
                            "fe80::3"
                        ]
                    }
                ]
            }]
        }
    }`, "2.2.0")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Found 1 affected pool:")
	require.Contains(t, *report.content, "configuration contains address "+
		"pools with the number of in-pool IP reservations equal to their size")
	require.Contains(t, *report.content,
		"1. Pool 'fe80::1-fe80::3' of the 'fe80::/16' subnet")
	require.NotContains(t, *report.content, "2.")
}

// Test that only the first 10 affected pools are included in the report.
func TestAddressPoolsExhaustedByReservationsForMoreAffectedPoolsThanLimit(t *testing.T) {
	// Arrange
	subnets := []map[string]any{}
	// Generate 2 subnet.
	for s := 0; s <= 1; s++ {
		pools := []map[string]any{}
		reservations := []map[string]any{}
		// Each subnet included 7 single-address pools and 7 reservations.
		for a := 1; a <= 7; a++ {
			address := fmt.Sprintf("10.0.%d.%d", s, a)

			pool := map[string]any{
				"pool": fmt.Sprintf("%s-%s", address, address),
			}
			pools = append(pools, pool)

			reservation := map[string]any{
				"ip-address": address,
			}
			reservations = append(reservations, reservation)
		}

		subnets = append(subnets, map[string]any{
			"subnet":       fmt.Sprintf("10.0.%d.0/24", s),
			"pools":        pools,
			"reservations": reservations,
		})
	}

	config := map[string]any{
		"Dhcp4": map[string]any{
			"subnet4": subnets,
		},
	}
	configJSON, _ := json.Marshal(config)

	ctx := createReviewContext(t, nil, string(configJSON), "2.2.0")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "First 10 affected pools:")
	require.Contains(t, *report.content, "configuration contains address "+
		"pools with the number of in-pool IP reservations equal to their size")
	require.Contains(t, *report.content,
		"1. Pool '10.0.0.1-10.0.0.1' of the '10.0.0.0/24' subnet")
	require.Contains(t, *report.content,
		"8. Pool '10.0.1.1-10.0.1.1' of the '10.0.1.0/24' subnet")
	require.Contains(t, *report.content, "\n10.")
	require.NotContains(t, *report.content, "\n11.")
	require.NotContains(t, *report.content,
		"Pool '10.0.1.7-10.0.1.7' of the '10.0.1.0/24' subnet")
}

// Test that the report contains the subnet IDs if provided.
func TestAddressPoolsExhaustedByReservationsReportContainsSubnetID(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "id": 42,
                "subnet": "fe80::/16",
                "pools": [{ "pool": "fe80::1-fe80::1" }],
                "reservations": [{ "ip-address": "fe80::1" }]
            }]
        }
    }`, "2.2.0")
	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"1. Pool 'fe80::1-fe80::1' of the '[42] fe80::/16' subnet")
}

// Test that the IP reservations from the database are considered when checking
// if the pool is exhausted.
func TestAddressPoolsExhaustedByReservationsConsidersDatabaseReservations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	config := `{
        "Dhcp6": {
            "hooks-libraries": [ {
                "library": "/usr/lib/kea/libdhcp_host_cmds.so"
            } ],
            "subnet6": [{
                "id": 42,
                "subnet": "fe80::/16",
                "pools": [{ "pool": "fe80::1-fe80::1" }]
            }]
        }
    }`

	ctx := createReviewContext(t, db, config, "2.2.0")

	createHostInDatabase(t, db, config, "fe80::/16", "fe80::1")

	// Act
	report, err := addressPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"1. Pool 'fe80::1-fe80::1' of the '[42] fe80::/16' subnet")
}

// Test that the error is returned if the non-DHCP daemon is checked.
func TestDelegatedPrefixPoolsExhaustedByReservationsForNonDHCPDaemonConfig(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Control-agent": { }
    }`, "2.2.0")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.Nil(t, report)
	require.ErrorContains(t, err, "unsupported daemon")
}

// Test that the no error and no issue are returned if the configuration
// doesn't contain subnets.
func TestDelegatedPrefixPoolsExhaustedByReservationsForMissingSubnets(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp4": {}
    }`, "2.2.0")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.Nil(t, report)
	require.Nil(t, err)
}

// Test that no issue report is returned if the number of reservations is
// less then the number of available prefixes in the pool.
func TestDelegatedPrefixPoolsExhaustedByReservationsForLessReservationsThanAddresses(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "subnet": "fe80::/16",
                "pd-pools": [
                    {
                        "prefix": "fe80::",
                        "prefix-len": 64,
                        "delegated-len": 80
                    }
                ],
                "reservations": [
                    {
                        "prefixes": [
                            "fe80:1::/96",
                            "fe80:2::/96"
                        ]
                    }
                ]
            }]
        }
    }`, "2.2.0")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the issue report is returned if all pool prefixes are reserved.
func TestDelegatedPrefixPoolsExhaustedByReservationsForEqualReservationsAndAddresses(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "subnet": "fe80::/16",
                "pd-pools": [
                    {
                        "prefix": "fe80::",
                        "prefix-len": 125,
                        "delegated-len": 127
                    }
                ],
                "reservations": [
                    {
                        "prefixes": [
                            "fe80::0/127",
                            "fe80::2/127",
                            "fe80::4/127"
                        ]
                    },
                    {
                        "prefixes": [
                            "fe80::6/127"
                        ]
                    }
                ]
            }]
        }
    }`, "2.2.0")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Found 1 affected pool:")
	require.Contains(t, *report.content, "configuration contains delegated "+
		"prefix pools with the number of in-pool PD reservations equal to their size")
	require.Contains(t, *report.content,
		"1. Pool 'fe80::/125 del. 127' of the 'fe80::/16' subnet")
	require.NotContains(t, *report.content, "2.")
}

// Test that only the first 10 affected pools are included in the report.
func TestDelegatedPrefixPoolsExhaustedByReservationsForMoreAffectedPoolsThanLimit(t *testing.T) {
	// Arrange
	subnets := []map[string]any{}
	// Generate 2 subnet.
	for s := 0; s <= 1; s++ {
		pools := []map[string]any{}
		reservations := []map[string]any{}
		// Each subnet included 7 single-prefix pools and 7 reservations.
		for a := 1; a <= 7; a++ {
			prefix := fmt.Sprintf("fe80::%d:%d", s, a)

			pool := map[string]any{
				"prefix":        prefix,
				"prefix-len":    127,
				"delegated-len": 127,
			}
			pools = append(pools, pool)

			reservation := map[string]any{
				"prefixes": []string{
					fmt.Sprintf("%s/127", prefix),
				},
			}
			reservations = append(reservations, reservation)
		}

		subnets = append(subnets, map[string]any{
			"subnet":       fmt.Sprintf("fe80::%d:0/112", s),
			"pd-pools":     pools,
			"reservations": reservations,
		})
	}

	config := map[string]any{
		"Dhcp6": map[string]any{
			"subnet6": subnets,
		},
	}
	configJSON, _ := json.Marshal(config)

	ctx := createReviewContext(t, nil, string(configJSON), "2.2.0")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "First 10 affected pools:")
	require.Contains(t, *report.content, "configuration contains delegated prefix "+
		"pools with the number of in-pool PD reservations equal to their size")
	require.Contains(t, *report.content,
		"1. Pool 'fe80::0:1/127 del. 127' of the 'fe80::0:0/112' subnet")
	require.Contains(t, *report.content,
		"8. Pool 'fe80::1:1/127 del. 127' of the 'fe80::1:0/112' subnet")
	require.Contains(t, *report.content, "\n10.")
	require.NotContains(t, *report.content, "\n11.")
	require.NotContains(t, *report.content,
		"Pool 'fe80::1:4/127 del. 127' of the 'fe80::1:0/112' subnet")
}

// Test that the report contains the subnet IDs if provided.
func TestDelegatedPrefixPoolsExhaustedByReservationsReportContainsSubnetID(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{
        "Dhcp6": {
            "subnet6": [{
                "id": 42,
                "subnet": "fe80::/16",
                "pd-pools": [{ "prefix": "fe80::", "prefix-len": 80, "delegated-len": 80 }],
                "reservations": [{ "prefixes": ["fe80::/80"] }]
            }]
        }
    }`, "2.2.0")
	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"1. Pool 'fe80::/80 del. 80' of the '[42] fe80::/16' subnet")
}

// Test that the IP reservations from the database are considered when checking
// if the pool is exhausted.
func TestDelegatedPrefixPoolsExhaustedByReservationsConsidersDatabaseReservations(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	config := `{
        "Dhcp6": {
            "hooks-libraries": [ {
                "library": "/usr/lib/kea/libdhcp_host_cmds.so"
            } ],
            "subnet6": [{
                "id": 42,
                "subnet": "fe80::/16",
                "pd-pools": [{ "prefix": "fe80::", "prefix-len": 80, "delegated-len": 80 }]
            }]
        }
    }`

	ctx := createReviewContext(t, db, config, "2.2.0")

	createHostInDatabase(t, db, config, "fe80::/16", "fe80::/80")

	// Act
	report, err := delegatedPrefixPoolsExhaustedByReservations(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content,
		"1. Pool 'fe80::/80 del. 80' of the '[42] fe80::/16' subnet")
}

// Test that the checker returns an error if provided a non-DHCP daemon.
func TestSubnetCmdsAndConfigBackendMutualExclusionForNonDHCPDaemon(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {} }`, "2.2.0")
	// Act
	report, err := subnetCmdsAndConfigBackendMutualExclusion(ctx)

	// Assert
	require.ErrorContains(t, err, "unsupported daemon")
	require.Nil(t, report)
}

// Test that the checker founds no issue if the subnet hook is missing.
func TestSubnetCmdsAndConfigBackendMutualExclusionForMissingSubnetHook(t *testing.T) {
	// Arrange
	configStr := `{
        "Dhcp4": {
            "config-control": {
                "config-databases": [
                    {
                        "name": "config",
                        "type": "mysql"
                    }
                ]
            }
        }
    }`
	ctx := createReviewContext(t, nil, configStr, "2.2.0")

	// Act
	report, err := subnetCmdsAndConfigBackendMutualExclusion(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the checker founds no issue if no config backend databases are
// configured.
func TestSubnetCmdsAndConfigBackendMutualExclusionForMissingConfigBackend(t *testing.T) {
	// Arrange
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_subnet_cmds.so"
                }
            ]
        }
    }`
	ctx := createReviewContext(t, nil, configStr, "2.2.0")

	// Act
	report, err := subnetCmdsAndConfigBackendMutualExclusion(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the checker founds an issue if the subnet hook and the config
// backend database are used mutually.
func TestSubnetCmdsAndConfigBackendMutualExclusionDetection(t *testing.T) {
	// Arrange
	configStr := `{
        "Dhcp6": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_subnet_cmds.so"
                }
            ],
            "config-control": {
                "config-databases": [
                    {
                        "name": "config",
                        "type": "mysql"
                    }
                ]
            }
        }
    }`
	ctx := createReviewContext(t, nil, configStr, "2.2.0")

	// Act
	report, err := subnetCmdsAndConfigBackendMutualExclusion(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.EqualValues(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(
		t,
		*report.content,
		"is recommended that the 'subnet_cmds' hook library not be used "+
			"to manage subnets when the configuration backend is used",
	)
}

// Test that the credentials over HTTPS checker returns an error for the
// not-CA daemons.
func TestCredentialsOverHTTPSForNonCADaemon(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": { } }`, "2.2.0")

	// Act
	report, err := credentialsOverHTTPS(ctx)

	// Assert
	require.Nil(t, report)
	require.ErrorContains(t, err, "unsupported daemon")
}

// Test that the credentials over HTTPS checker returns no report if the
// basic authentication is not used.
func TestCredentialsOverHTTPSForNoBasicAuth(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": { } }`, "2.2.0")

	// Act
	report, err := credentialsOverHTTPS(ctx)

	// Assert
	require.Nil(t, report)
	require.NoError(t, err)
}

// Test that the credentials over HTTPS checker reports an issue if the HTTP
// credentials are provided but the Stork agent and Kea Control Agent don't
// communicate over the secure protocol.
func TestCredentialsOverHTTPSForProvidedCredentialsWithoutTLS(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {
		"authentication": {
			"type": "basic",
			"realm": "kea-dhcp-ddns-server",
			"clients": [{ "user": "admin", "password": "1234" }]
		}
	} }`, "2.2.0")

	// Act
	report, err := credentialsOverHTTPS(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t, *report.content, "Configure the 'trust-anchor', "+
		"'cert-file', and 'key-file' properties in the Kea Control Agent "+
		"{daemon} configuration to use the secure protocol.")
}

// Test that the credentials over HTTPS checker reports no issue if the HTTP
// credentials are provided and the Stork agent and Kea Control Agent
// communicate over the secure protocol.
func TestCredentialsOverHTTPSForProvidedCredentialsWithTLS(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {
        "trust-anchor": "foo",
        "cert-file": "/bar",
        "key-file": "/baz",
		"authentication": {
			"type": "basic",
			"realm": "kea-dhcp-ddns-server",
			"clients": [{ "user": "admin", "password": "1234" }]
		}
    } }`, "2.2.0")

	// Act
	report, err := credentialsOverHTTPS(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the control sockets CA checker reports an issue if the control
// sockets entry is missing in the Kea Control Agent configuration.
func TestControlSocketsCAMissingEntry(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {} }`, "2.2.0")

	// Act
	report, err := controlSocketsCA(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t,
		*report.content,
		"The control sockets are not specified in the Kea Control Agent",
	)
}

// Test that the control sockets CA checker reports an issue if the control
// sockets entry is provided in the Kea Control Agent configuration but it has
// no configured daemons.
func TestControlSocketsCAEmptyEntry(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {
		"control-sockets": { }
	} }`, "2.2.0")

	// Act
	report, err := controlSocketsCA(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t,
		*report.content,
		"The control sockets entry in the Kea Control Agent {daemon} configuration is empty.",
	)
}

// Test that the control sockets CA checker reports an issue if the control
// sockets entry is provided in the Kea Control Agent configuration but it lacks
// any DHCP daemon.
func TestControlSocketsCAMissingDHCPDaemons(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {
		"control-sockets": {
			"d2": {
				"socket-type": "unix",
				"socket-name": "/path/to/the/unix/socket-d2"
			}
		}
	} }`, "2.2.0")

	// Act
	report, err := controlSocketsCA(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
	require.Contains(t,
		*report.content,
		"The control sockets entry in the Kea Control Agent {daemon} configuration doesn't contain path to any DHCP daemon",
	)
}

// Test that the control sockets CA checker reports no issue if the control
// sockets entry is provided in the Kea Control Agent configuration and it has
// at least one configured DHCP daemon.
func TestControlSocketsCAProperConfig(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": {
		"control-sockets": {
			"dhcp4": {
				"socket-type": "unix",
				"socket-name": "/tmp/kea4-ctrl-socket"
			}
		}
	} }`, "2.2.0")

	// Act
	report, err := controlSocketsCA(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the control sockets CA checker returns an error for the
// not-CA daemons.
func TestControlSocketsCAForNonCADaemon(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp4": { } }`, "2.2.0")

	// Act
	report, err := controlSocketsCA(ctx)

	// Assert
	require.Nil(t, report)
	require.ErrorContains(t, err, "unsupported daemon")
}

// Test that the checker for the unavailable gathering statics capabilities
// doesn't support non-DHCP daemons.
func TestGatheringStatsUnavailableForNonDHCPDaemon(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Control-agent": { } }`, "2.2.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.ErrorContains(t, err, "unsupported daemon")
	require.Nil(t, report)
}

// Test that the checker for the unavailable gathering statics capabilities
// does nothing if the stats hook is missing.
func TestGatheringStatsUnavailableForMissingStatsHook(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pools": [ { "pool": "fe80::1-fe80:ffff:ffff:ffff:ffff:ffff:ffff:fff" } ]
		} ]
	} }`, "2.2.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the checker for the unavailable gathering statics capabilities
// does nothing if the stat hook is present and the Kea version is 2.5.3 or
// above.
func TestGatheringStatsUnavailableForKea253OrAbove(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pools": [ { "pool": "fe80::1-fe80:ffff:ffff:ffff:ffff:ffff:ffff:fff" } ]
		} ]
	} }`, "2.5.3")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the checker for the unavailable gathering statics capabilities
// detects the overflow in the number of addresses of the shared network.
func TestGatheringStatsUnavailableForOverflowingSharedNetwork(t *testing.T) {
	// Arrange
	// Shared network with 2^112-1 addresses.
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"shared-networks": [ {
			"name": "foo",
			"subnet6": [ {
				"subnet": "fe80::/16",
				"pools": [ { "pool": "fe80::1-fe80:ffff:ffff:ffff:ffff:ffff:ffff:fff" } ]
			} ]
		} ]
	} }`, "2.4.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
}

// Test that the checker for the unavailable gathering statics capabilities
// detects the overflow in the number of addresses of the global subnets.
func TestGatheringStatsUnavailableForOverflowingGlobalSubnets(t *testing.T) {
	// Arrange
	// Global subnet with 2^112-1 addresses.
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pools": [ { "pool": "fe80::1-fe80:ffff:ffff:ffff:ffff:ffff:ffff:fff" } ]
		} ]
	} }`, "2.4.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
}

// Test that the checker for the unavailable gathering statics capabilities
// detects the overflow in the number of delegated prefixes of the shared
// networks.
func TestGatheringStatsUnavailableForOverflowingSharedNetworkDelegatedPrefixes(t *testing.T) {
	// Arrange
	// Shared network with 2^63 prefixes.
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"shared-networks": [ {
			"name": "foo",
			"subnet6": [ {
				"subnet": "fe80::/16",
				"pd-pools": [ { "prefix": "fe80::", "prefix-len": 64, "delegated-len": 127 } ]
			} ]
		} ]
	} }`, "2.4.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
}

// Test that the checker for the unavailable gathering statics capabilities
// detects the overflow in the number of delegated prefixes of the global
// subnets.
func TestGatheringStatsUnavailableForOverflowingGlobalDelegatedPrefixes(t *testing.T) {
	// Arrange
	// Global subnet with 2^63 prefixes.
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pd-pools": [ { "prefix": "fe80::", "prefix-len": 64, "delegated-len": 127 } ]
		} ]
	} }`, "2.4.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, ctx.subjectDaemon.ID, report.daemonID)
	require.Len(t, report.refDaemonIDs, 1)
	require.Contains(t, report.refDaemonIDs, ctx.subjectDaemon.ID)
	require.NotNil(t, report.content)
}

// Test that the checker for the unavailable gathering statics capabilities
// returns no report and no error if there is no overflow in the number of
// addresses or delegated prefixes.
func TestGatheringStatsUnavailableForNonOverflowingConfig(t *testing.T) {
	// Arrange
	ctx := createReviewContext(t, nil, `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pools": [ { "pool": "fe80::1-fe80::2" } ],
			"pd-pools": [ { "prefix": "fe80::", "prefix-len": 64, "delegated-len": 64 } ]
		} ],
		"shared-networks": [ {
			"name": "foo",
			"subnet6": [ {
				"subnet": "fe81::/16",
				"pools": [ { "pool": "fe81::1-fe81::2" } ],
				"pd-pools": [ { "prefix": "fe81::", "prefix-len": 64, "delegated-len": 64 } ]
			} ]
		} ]
	} }`, "2.4.0")

	// Act
	report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report)
}

// Test that the checker for the unavailable gathering statics capabilities
// produces a different report depending on the Kea version.
func TestGatheringStatsUnavailableReportForDifferentKeaVersions(t *testing.T) {
	// Arrange
	config := `{ "Dhcp6": {
		"hooks-libraries": [ { "library": "/usr/lib/kea/libdhcp_stat_cmds.so" } ],
		"subnet6": [ {
			"subnet": "fe80::/16",
			"pools": [ { "pool": "fe80::1-fe80:ffff:ffff:ffff:ffff:ffff:ffff:fff" } ]
		} ]
	} }`

	for major := 1; major < 5; major++ {
		for minor := 0; minor < 5; minor++ {
			for patch := 0; patch < 20; patch++ {
				version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
				semanticVersion := storkutil.NewSemanticVersion(major, minor, patch)

				t.Run(version, func(t *testing.T) {
					ctx := createReviewContext(t, nil, config, version)

					// Act
					report, err := gatheringStatisticsUnavailableDueToNumberOverflow(ctx)

					// Assert
					if semanticVersion.GreaterThan(storkutil.NewSemanticVersion(2, 5, 3)) {
						// Patched version. No issue should be reported.
						require.NoError(t, err)
						require.Nil(t, report)
						return
					}

					require.NoError(t, err)
					require.NotNil(t, report)

					expectedMessages := []string{
						// Common report part.
						"The Kea {daemon} daemon has configured some very large pools.",
						"the 'fe80::/16' subnet has more than 2^63-1 addresses.",
					}

					// Version-specific part.
					if semanticVersion.LessThan(storkutil.NewSemanticVersion(2, 3, 0)) {
						expectedMessages = append(expectedMessages,
							"The statistics presented by Stork "+
								"and Prometheus/Grafana may be inaccurate",
						)
					} else {
						expectedMessages = append(expectedMessages,
							"Stork is unable to fetch them.",
						)
					}

					for _, message := range expectedMessages {
						require.Contains(t, *report.content, message)
					}
				})
			}
		}
	}
}

// Test that the overflow is detected if the number of addresses in the pool
// exceeds the limit.
func TestFindSharedNetworkExceedingAddressLimitForSingleOverflowingSubnetPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					Pool: "fe80::1 - fe80:f:ffff:ffff:ffff:ffff:ffff:ffff",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'fe80::/16' subnet has more than 2^63-1 addresses")
}

// Test that the overflow is not detected if the number of addresses in any
// pool of global subnets doesn't exceed the limit.
func TestFindSharedNetworkExceedingAddressLimitForSingleNonOverflowingGlobalSubnetPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					Pool: "fe80::1 - fe80::2",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Name:    "foo",
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.False(t, isOverflow)
	require.Empty(t, reason)
}

// Test that the overflow is not detected if the number of addresses in any
// pool of shared networks doesn't exceed the limit.
func TestFindSharedNetworkExceedingAddressLimitForSingleNonOverflowingSharedNetworkPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					Pool: "fe80::1 - fe80::2",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.False(t, isOverflow)
	require.Empty(t, reason)
}

// Test that the overflow is detected even if the number of addresses in the
// particular pools doesn't exceed the limit but the total number of addresses
// in the subnet does.
func TestFindSharedNetworkExceedingAddressLimitForOverflowingSubnet(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					// 2^63-1 addresses.
					Pool: "fe80::1 - fe80::7fff:ffff:ffff:ffff",
				},
				{
					// 1 address.
					Pool: "fe80:42::1 - fe80:42::1",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'fe80::/16' subnet has more than 2^63-1 addresses")
}

// Test that the overflow is detected even if the number of addresses in the
// particular subnets doesn't exceed the limit but the total number of
// addresses in the shared network does.
func TestFindSharedNetworkExceedingAddressLimitForOverflowingSharedNetwork(t *testing.T) {
	// Arrange
	subnet1 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					// 2^63-1 addresses.
					Pool: "fe80::1 - fe80::7fff:ffff:ffff:ffff",
				},
			},
		},
	}

	subnet2 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					// 1 address.
					Pool: "fe80:42::1 - fe80:42::1",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Name:    "foo",
			Subnet6: []keaconfig.Subnet6{subnet1, subnet2},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'foo' shared network has more than 2^63-1 addresses")
}

// Test that the overflow is not detected even if the total number of
// addresses of the top subnets exceed the limit.
func TestFindSharedNetworkExceedingAddressLimitForOverflowingGlobalSubnets(t *testing.T) {
	// Arrange
	subnet1 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					// 2^63-1 addresses.
					Pool: "fe80::1 - fe80::7fff:ffff:ffff:ffff",
				},
			},
		},
	}

	subnet2 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					// 1 address.
					Pool: "fe80:42::1 - fe80:42::1",
				},
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			// Global subnets. Shared network name is empty.
			Subnet6: []keaconfig.Subnet6{subnet1, subnet2},
		},
	}

	// Act
	isOverflow, reason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)

	// Assert
	require.NoError(t, err)
	require.False(t, isOverflow)
	require.Empty(t, reason)
}

// Test that the overflow is detected if the number of delegated prefixes
// in the pool exceeds the limit.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForSingleOverflowingSubnetPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 2^63 delegated prefixes.
				PrefixLen:    64,
				DelegatedLen: 127,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'fe80::/16' subnet has more than 2^63-1 delegated prefixes")
}

// Test that the overflow is not detected if the number of delegated prefixes
// in any global subnet pool doesn't exceed the limit.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForSingleNonOverflowingGlobalSubnetPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 1 delegated prefix.
				PrefixLen:    64,
				DelegatedLen: 64,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.False(t, isOverflow)
	require.Empty(t, reason)
}

// Test that the overflow is not detected if the number of delegated prefixes
// in any shared network pool doesn't exceed the limit.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForSingleNonOverflowingSharedNetworkPool(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 1 delegated prefix.
				PrefixLen:    64,
				DelegatedLen: 64,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Name:    "foo",
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.False(t, isOverflow)
	require.Empty(t, reason)
}

// Test that the overflow is detected even if the number of delegated prefixes
// in the particular pools doesn't exceed the limit but the total number of
// delegated prefixes in the subnet does.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForOverflowingSubnet(t *testing.T) {
	// Arrange
	subnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 2^62 delegated prefixes.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
			{
				Prefix: "fe80:42::",
				// 2^62 delegated prefix.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Subnet6: []keaconfig.Subnet6{subnet},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'fe80::/16' subnet has more than 2^63-1 delegated prefixes")
}

// Test that the overflow is detected even if the number of delegated prefixes
// in the particular subnets doesn't exceed the limit but the total number of
// delegated prefixes in the shared network does.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForOverflowingSharedNetwork(t *testing.T) {
	// Arrange
	subnet1 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 2^62 delegated prefixes.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
		},
	}

	subnet2 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80:42::",
				// 2^62 delegated prefix.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			Name:    "foo",
			Subnet6: []keaconfig.Subnet6{subnet1, subnet2},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.True(t, isOverflow)
	require.Contains(t, reason, "the 'foo' shared network has more than 2^63-1 delegated prefixes")
}

// Test that the overflow is not detected even if the total number of
// delegated prefixes of the top subnets exceed the limit.
func TestFindSharedNetworkExceedingDelegatedPrefixLimitForOverflowingGlobalSubnets(t *testing.T) {
	// Arrange
	subnet1 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80::",
				// 2^62 delegated prefixes.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
		},
	}

	subnet2 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::/16",
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix: "fe80:42::",
				// 2^62 delegated prefix.
				PrefixLen:    64,
				DelegatedLen: 126,
			},
		},
	}

	sharedNetworks := []keaconfig.SharedNetwork{
		keaconfig.SharedNetwork6{
			// Global subnets. Shared network name is empty.
			Subnet6: []keaconfig.Subnet6{subnet1, subnet2},
		},
	}

	// Act
	isOverflow, reason := findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)

	// Assert
	require.False(t, isOverflow)
	require.Empty(t, reason)
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
		}, Triggers{ManualRun}, nil)
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
			LocalHosts: []dbmodel.LocalHost{
				{
					DaemonID:   app.Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI,
					IPReservations: []dbmodel.IPReservation{
						{
							Address: fmt.Sprintf("%s.5", prefix),
						},
					},
				},
			},
		}
		// Add the host.
		err = dbmodel.AddHost(db, host)
		if err != nil {
			b.Fatalf("failed to add app to subnet %s: %+v", dbSubnet.Prefix, err)
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
		}, Triggers{ManualRun}, nil)
		_, err = reservationsOutOfPool(ctx)
		if err != nil {
			b.Fatalf("checker failed: %+v", err)
		}
	}
}

// Generates subnets of which some have overlapping prefixes.
// The overlapping factor must be in range from 0 (no overlaps) to 1 (100% overlaps).
// Each overlapped subnet is contained in exactly one other subnet.
func getOverlappingSubnets(n int, overlappingFactor float32) (subnets []keaconfig.Subnet) {
	overlappingStep := int(float32(n) * overlappingFactor)

	for i := 0; i < n; i++ {
		id := int64(i + 1)
		index := i
		mask := 24

		if overlappingFactor != 0. && i%overlappingStep == 1 {
			index--
			mask++
		}

		part4 := 0
		part3 := index % 256
		part2 := (index / 256) % 256
		part1 := (index / (256 * 256)) % 256

		prefix := fmt.Sprintf("%d.%d.%d.%d/%d", part1, part2, part3, part4, mask)

		subnet := keaconfig.Subnet4{
			MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
				ID:     id,
				Subnet: prefix,
			},
		}
		subnets = append(subnets, &subnet)
	}

	return subnets
}

// Measures the performance of the overlapping prefixes detection based on the
// binary prefixes without using the radix tree.
// The possible solutions were discussed in this thread:
// https://gitlab.isc.org/isc-projects/stork/-/merge_requests/474#note_305555
func BenchmarkOverlapsBinaryPrefixesOnly(b *testing.B) {
	numberOfSubnets := 8196
	overlappingFactor := float32(0.01)
	maximumOverlaps := 10

	subnets := getOverlappingSubnets(numberOfSubnets, overlappingFactor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = findOverlaps(subnets, maximumOverlaps)
	}
}
