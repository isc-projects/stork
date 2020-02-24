package kea

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Creates an app instance used in the tests. The index value should be incremented
// for each new app to make sure that the address/address port tuple inserted to
// the database is unique. The DHCPv4 and DHCPv6 configurations provided as text.
// If any of them is empty, it is ignored. The created app instance is inserted
// to the database and then returned to the unit test.
func createAppWithSubnets(t *testing.T, db *dbops.PgDB, index int64, v4Config, v6Config string) *dbmodel.App {
	// Add the machine.
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080 + index,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// DHCPv4 configuration.
	var kea4Config *dbmodel.KeaConfig
	if len(v4Config) > 0 {
		kea4Config, err = dbmodel.NewKeaConfigFromJSON(v4Config)
		require.NoError(t, err)
	}

	// DHCPv6 configuration.
	var kea6Config *dbmodel.KeaConfig
	if len(v6Config) > 0 {
		kea6Config, err = dbmodel.NewKeaConfigFromJSON(v6Config)
		require.NoError(t, err)
	}

	// Creates new app with provided configurations.
	app := dbmodel.App{
		MachineID:   m.ID,
		Type:        dbmodel.KeaAppType,
		CtrlAddress: "localhost",
		CtrlPort:    8000,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{
				{
					Name:   "dhcp4",
					Config: kea4Config,
				},
				{
					Name:   "dhcp6",
					Config: kea6Config,
				},
			},
		},
	}
	// Add the app to the database.
	err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	return &app
}

// Multi step test which verifies that the subnets and shared networks can be
// created and matched when new Kea app instances are being added.
func TestDetectNetworks(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	v4Config := `
        {
            "Dhcp4": {
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24"
                    },
                    {
                        "subnet": "192.0.3.0/24"
                    }
                ]
            }   
        }`

	v6Config := `
        {
            "Dhcp6": {
                "subnet6": [
                    {
                        "subnet": "2001:db8:1::/64"
                    },
                    {
                        "subnet": "2001:db8:2::/64"
                    }
                ]
            }   
        }`
	app := createAppWithSubnets(t, db, 0, v4Config, v6Config)

	// First case: there are no subnets nor shared networks in the database.
	networks, subnets, err := DetectNetworks(db, app)
	require.NoError(t, err)
	// The configuration lacks shared networks so they should not be
	// returned.
	require.Empty(t, networks)
	// There should be 4 new subnets returned as a result of processing
	// the configurations of this app.
	require.Len(t, subnets, 4)

	// Verify that the subnets are correct.
	for _, s := range subnets {
		// The app is not associated automatically with the subnets. This is
		// to indicate that the subnet is not associated with the app until
		// explicitly requested.
		require.Empty(t, s.Apps)
		// The subnets haven't been added to the database yet. Their IDs should
		// be 0. That also allows to determine that this is a new subnet.
		require.Zero(t, s.ID)

		// Ok let's add the subnet to the db and associate this subnet with
		// our app.
		subnet := s
		err = dbmodel.AddSubnet(db, &subnet)
		require.NoError(t, err)
		err = dbmodel.AddAppToSubnet(db, &subnet, app)
		require.NoError(t, err)
	}

	// Second case: introducing a shared network. Note that the top level subnet
	// already exists in the database.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.0.0.0/8"
                            }
                        ]
                    }
                ],
                "subnet4": [
                    {
                        "subnet": "192.0.2.0/24"
                    }
                ]
            }
        }`
	app = createAppWithSubnets(t, db, 1, v4Config, "")

	networks, subnets, err = DetectNetworks(db, app)
	require.NoError(t, err)
	// This time we should get one shared network in return.
	require.Len(t, networks, 1)
	newNetwork := networks[0]

	// This is new shared network so its ID should be 0 until explicitly
	// added to the database.
	require.Zero(t, newNetwork.ID)
	require.Equal(t, "foo", newNetwork.Name)
	// The shared network contains one subnet.
	require.Len(t, newNetwork.Subnets, 1)
	// This is new subnet so the ID should be unset.
	require.Zero(t, newNetwork.Subnets[0].ID)
	require.Equal(t, "10.0.0.0/8", newNetwork.Subnets[0].Prefix)
	// The subnet is not associated with any apps until such association
	// is explicitly made.
	require.Empty(t, newNetwork.Subnets[0].Apps)

	// Add the shared network and the subnet it contains to the database.
	err = dbmodel.AddSharedNetwork(db, &newNetwork)
	require.NoError(t, err)

	// The subnet ID should have been set after being added to the database.
	newSubnet := newNetwork.Subnets[0]
	require.NotZero(t, newSubnet.ID)

	// The association of the subnet and the app must be done explicitly.
	err = dbmodel.AddAppToSubnet(db, &newSubnet, app)
	require.NoError(t, err)

	// Verify that we have one top level subnet.
	require.Len(t, subnets, 1)
	newSubnet = subnets[0]

	// This subnet was already present in the database, therefore it should
	// already have an ID.
	require.NotZero(t, newSubnet.ID)
	require.Equal(t, "192.0.2.0/24", newSubnet.Prefix)
	// Also this subnet should be already associated with the previous app.
	require.Len(t, newSubnet.Apps, 1)

	// Add association of our new app with that subnet.
	err = dbmodel.AddAppToSubnet(db, &newSubnet, app)
	require.NoError(t, err)

	// Third case: two shared networks of which one already exists.
	v4Config = `
        {
            "Dhcp4": {
                "shared-networks": [
                    {
                        "name": "foo",
                        "subnet4": [
                            {
                                "subnet": "10.0.0.0/8"
                            },
                            {
                                "subnet": "10.1.0.0/16"
                            }
                        ]
                    },
                    {
                        "name": "bar",
                        "subnet4": [
                            {
                                "subnet": "192.0.3.0/24"
                            },
                            {
                                "subnet": "192.0.4.0/24"
                            }
                        ]
                    }
                ]
            }   
        }`
	app = createAppWithSubnets(t, db, 2, v4Config, "")

	networks, subnets, err = DetectNetworks(db, app)
	require.NoError(t, err)
	require.Len(t, networks, 2)
	// This time there should be no top level subnets.
	require.Empty(t, subnets)

	// First shared network was already in the database so it should have
	// non zero id.
	require.NotZero(t, networks[0].ID)
	require.Equal(t, "foo", networks[0].Name)
	// It should now include two subnets: one old and one new.
	require.Len(t, networks[0].Subnets, 2)

	// The first subnet already existed in the database so it should
	// have non zero id.
	require.NotZero(t, networks[0].Subnets[0].ID)
	require.Equal(t, "10.0.0.0/8", networks[0].Subnets[0].Prefix)
	// Also, this subnet already had an association with one of the apps.
	require.Len(t, networks[0].Subnets[0].Apps, 1)

	// The second subnet is new and therefore has id of 0.
	require.Zero(t, networks[0].Subnets[1].ID)
	require.Equal(t, "10.1.0.0/16", networks[0].Subnets[1].Prefix)
	// Also, it is not associated with any apps yet.
	require.Empty(t, networks[0].Subnets[1].Apps)

	// The second shared network is brand new. The subnets in it are
	// also brand new.
	require.Equal(t, "bar", networks[1].Name)
	require.Len(t, networks[1].Subnets, 2)
	require.Zero(t, networks[1].Subnets[0].ID)
	require.Equal(t, "192.0.3.0/24", networks[1].Subnets[0].Prefix)
	require.Empty(t, networks[1].Subnets[0].Apps)
	require.Zero(t, networks[1].Subnets[1].ID)
	require.Equal(t, "192.0.4.0/24", networks[1].Subnets[1].Prefix)
	require.Empty(t, networks[1].Subnets[1].Apps)
}
