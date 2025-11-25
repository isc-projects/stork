package dbops_test

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/daemons/kea"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
)

// Current schema version. This value must be bumped up every
// time the schema is updated.
const expectedSchemaVersion int64 = 71

// Common function which tests a selected migration action.
func testMigrateAction(t *testing.T, db *dbops.PgDB, expectedOldVersion, expectedNewVersion int64, action ...string) {
	oldVersion, newVersion, err := dbops.Migrate(db, action...)
	require.NoError(t, err)

	// Check that old database version has been returned as expected.
	require.Equal(t, expectedOldVersion, oldVersion)

	// Check that new database version has been returned as expected.
	require.Equal(t, expectedNewVersion, newVersion)
}

// Checks that schema version can be fetched from the database and
// that it is set to an expected value.
func testCurrentVersion(t *testing.T, db *dbops.PgDB, expected int64) {
	current, err := dbops.CurrentVersion(db)
	require.NoError(t, err)
	require.Equal(t, expected, current)
}

// Test migrations between different database versions.
func TestMigrate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := dbops.Toss(db)
	require.NoError(t, err)

	// Create versioning table in the database.
	testMigrateAction(t, db, 0, 0, "init")
	// Migrate from version 0 to version 1.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Migrate from version 1 to version 0.
	testMigrateAction(t, db, 1, 0, "down")
	// Migrate to version 1 again.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Check current version.
	testMigrateAction(t, db, 1, 1, "version")
	// Reset to the initial version.
	testMigrateAction(t, db, 1, 0, "reset")
}

// Test initialization and migration in a single step.
func TestInitMigrate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := dbops.Toss(db)
	require.NoError(t, err)

	// Migrate from version 0 to version 1.
	testMigrateAction(t, db, 0, 1, "up", "1")
}

// Tests that the database schema can be initialized and migrated to the
// latest version with one call.
func TestInitMigrateToLatest(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := dbops.Toss(db)
	require.NoError(t, err)

	o, n, err := dbops.MigrateToLatest(db)
	require.NoError(t, err)
	require.Zero(t, o)
	require.GreaterOrEqual(t, n, int64(18))
}

// Test that available schema version is returned as expected.
func TestAvailableVersion(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := dbops.Toss(db)
	require.NoError(t, err)

	_, _, err = dbops.Migrate(db, "init")
	require.NoError(t, err)
	_, _, err = dbops.Migrate(db, "up")
	require.NoError(t, err)

	avail := dbops.AvailableVersion()
	require.Equal(t, avail, expectedSchemaVersion)
}

// Test that current version is returned from the database.
func TestCurrentVersion(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := dbops.Toss(db)
	require.NoError(t, err)

	// Initialize migrations.
	testMigrateAction(t, db, 0, 0, "init")
	// Initially, the version should be set to 0.
	testCurrentVersion(t, db, 0)
	// Go one version up.
	testMigrateAction(t, db, 0, 1, "up", "1")
	// Check that the current version is now set to 1.
	testCurrentVersion(t, db, 1)
}

// Test creating the server database and the user with access to
// this database using generated password.
func TestCreateDatabase(t *testing.T) {
	// Connect to the database with full privileges.
	_, dbSettings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	teardown()

	// Create a database and the user with the same name.
	dbName := fmt.Sprintf("storktest%d", rand.Int63())
	err := dbops.CreateDatabase(*dbSettings, dbName, dbName, "pass", true)
	require.NoError(t, err)

	// Try to connect to this database using the user name.
	opts := *dbSettings
	opts.User = dbName
	opts.Password = "pass"
	opts.DBName = dbName

	db2, err := dbops.NewPgDBConn(&opts)
	require.NoError(t, err)
	require.NotNil(t, db2)
	db2.Close()

	// Try to create the database again with the force flag and a different
	// password.
	err = dbops.CreateDatabase(*dbSettings, dbName, dbName, "pass2", true)
	require.NoError(t, err)

	// Attempt go create the database without the force flag should not
	// fail because the database already exists. The password is updated.
	err = dbops.CreateDatabase(*dbSettings, dbName, dbName, "pass3", false)
	require.NoError(t, err)

	// Connect to the database again using the second password.
	opts.Password = "pass3"

	db2, err = dbops.NewPgDBConn(&opts)
	require.NoError(t, err)
	require.NotNil(t, db2)
	defer db2.Close()

	// Check if the database has pgcrypto extension.
	hasExtension, err := maintenance.HasExtension(db2, "pgcrypto")
	require.NoError(t, err)
	require.True(t, hasExtension)
}

// Test that the pgcrypto database extension is successfully created.
func TestCreateCryptoExtension(t *testing.T) {
	// Connect to the database with full privileges.
	db, originalSettings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Create an empty database and the user with the same name.
	dbName := fmt.Sprintf("storktest%d", rand.Int63())
	_, err := maintenance.CreateDatabase(db, dbName)
	require.NoError(t, err)

	// Try to connect to this database using the user name.
	opts := *originalSettings
	opts.DBName = dbName
	db2, err := dbops.NewPgDBConn(&opts)
	require.NoError(t, err)
	require.NotNil(t, db2)

	// The new database should initially lack pgcrypto extension.
	hasExtension, err := maintenance.HasExtension(db2, "pgcrypto")
	require.NoError(t, err)
	require.False(t, hasExtension)

	// Create the pgcrypto extension.
	err = dbops.CreatePgCryptoExtension(db2)
	require.NoError(t, err)

	// Make sure the extension is now present.
	hasExtension, err = maintenance.HasExtension(db2, "pgcrypto")
	require.NoError(t, err)
	require.True(t, hasExtension)
}

// Test that the 39 migration convert decimals to bigints as back.
func TestMigration39DecimalToBigint(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dbops.Migrate(db, "down", "38")
	_, _ = db.Exec(`INSERT INTO statistic VALUES ('foo', NULL), ('bar', 42);`)

	expectedBigInt39, _ := big.NewInt(0).SetString(
		"12345678901234567890123456789012345678901234567890", 10,
	)
	expectedBigInt39Negative := big.NewInt(0).Mul(expectedBigInt39, big.NewInt(-1))

	// Act
	_, _, errUp := dbops.Migrate(db, "up", "39")
	_, errQuery := db.Exec(`
		INSERT INTO statistic VALUES
		('boz', '12345678901234567890123456789012345678901234567890'),
		('biz', '-12345678901234567890123456789012345678901234567890');
	`) // 50 digits
	stats39, errGet39 := dbmodel.GetAllStats(db)
	_, _, errDown := dbops.Migrate(db, "down", "38")
	stats38, errGet38 := dbmodel.GetAllStats(db)

	// Assert
	require.NoError(t, errUp)
	require.NoError(t, errQuery)
	require.NoError(t, errGet39)
	require.NoError(t, errDown)
	require.NoError(t, errGet38)

	require.Nil(t, stats39["foo"])
	require.EqualValues(t, big.NewInt(42), stats39["bar"])
	require.EqualValues(t, expectedBigInt39, stats39["boz"])
	require.EqualValues(t, expectedBigInt39Negative, stats39["biz"])

	require.Nil(t, stats38["foo"])
	require.EqualValues(t, big.NewInt(42), stats38["bar"])
	require.EqualValues(t, big.NewInt(math.MaxInt64), stats38["boz"])
	require.EqualValues(t, big.NewInt(math.MinInt64), stats38["biz"])
}

// Test that the 13 migration passes if some shared networks exist.
func TestMigration13AddInetFamilyColumn(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dbops.Migrate(db, "down", "12")

	var machineID int
	var appID int
	var sharedNetworkID int
	var subnetID int
	// Add an orphaned shared network.
	_, err := db.Exec(`INSERT INTO shared_network (name) VALUES ('frog');`)
	require.NoError(t, err)
	// Add a non-orphaned shared network.
	_, err = db.QueryOne(pg.Scan(&sharedNetworkID), `INSERT INTO shared_network (name) VALUES ('mouse') RETURNING id;`)
	require.NoError(t, err)
	_, err = db.QueryOne(pg.Scan(&machineID), `INSERT INTO machine (address, agent_port, state) VALUES ('foo', 42, '{}'::jsonb) RETURNING id;`)
	require.NoError(t, err)
	_, err = db.QueryOne(pg.Scan(&appID), `INSERT INTO app (machine_id, type) VALUES (?, 'kea') RETURNING id;`, machineID)
	require.NoError(t, err)
	_, err = db.QueryOne(pg.Scan(&subnetID), `INSERT INTO subnet (prefix, shared_network_id) VALUES ('fe80::/64', ?) RETURNING id;`, sharedNetworkID)
	require.NoError(t, err)

	// Act
	_, _, errUp := dbops.Migrate(db, "up", "13")

	// Assert
	require.NoError(t, errUp)
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM shared_network;`)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	var family int
	_, err = db.QueryOne(pg.Scan(&family), `SELECT inet_family FROM shared_network;`)
	require.NoError(t, err)
	require.EqualValues(t, 6, family)
}

// Asserts equality of two hosts list.
func assertHostsTheSame(t *testing.T, expected, actual []dbmodel.Host) {
	require.Equal(t, len(expected), len(actual))

	for i := range expected {
		expectedHost := expected[i]
		actualHost := actual[i]
		require.True(t, expectedHost.IsSame(&actualHost))

		// Check if the data after the down and up migrations are exactly equal
		// (except IDs).
		require.Equal(t, expectedHost.CreatedAt, actualHost.CreatedAt)
		require.Equal(t, expectedHost.HostIdentifiers, actualHost.HostIdentifiers)
		require.Equal(t, expectedHost.SubnetID, actualHost.SubnetID)
		require.Equal(t, len(expectedHost.LocalHosts), len(actualHost.LocalHosts))

		sort.Slice(expectedHost.LocalHosts, func(i, j int) bool {
			return expectedHost.LocalHosts[i].DaemonID < expectedHost.LocalHosts[j].DaemonID
		})
		sort.Slice(actualHost.LocalHosts, func(i, j int) bool {
			return actualHost.LocalHosts[i].DaemonID < actualHost.LocalHosts[j].DaemonID
		})

		for j := range expectedHost.LocalHosts {
			expectedLocalHost := expectedHost.LocalHosts[j]
			actualLocalHost := actualHost.LocalHosts[j]
			require.Equal(t, expectedLocalHost.Hostname, actualLocalHost.Hostname)
			require.Equal(t, expectedLocalHost.HostID, actualLocalHost.HostID)
			require.Equal(t, expectedLocalHost.DaemonID, actualLocalHost.DaemonID)
			require.Equal(t, expectedLocalHost.DataSource, actualLocalHost.DataSource)
			require.Equal(t, expectedLocalHost.BootFileName, actualLocalHost.BootFileName)
			require.Equal(t, expectedLocalHost.ClientClasses, actualLocalHost.ClientClasses)
			require.Equal(t, expectedLocalHost.DHCPOptionSet, actualLocalHost.DHCPOptionSet)
			require.Equal(t, expectedLocalHost.Hash, actualLocalHost.Hash)
			require.Equal(t, expectedLocalHost.NextServer, actualLocalHost.NextServer)
			require.Equal(t, expectedLocalHost.Options, actualLocalHost.Options)
			require.Equal(t, expectedLocalHost.ServerHostname, actualLocalHost.ServerHostname)
			require.Equal(t, len(expectedLocalHost.IPReservations), len(actualLocalHost.IPReservations))

			sort.Slice(expectedLocalHost.IPReservations, func(i, j int) bool {
				if expectedLocalHost.IPReservations[i].Address != expectedLocalHost.IPReservations[j].Address {
					return expectedLocalHost.IPReservations[i].Address < expectedLocalHost.IPReservations[j].Address
				}
				return expectedLocalHost.IPReservations[i].LocalHostID < expectedLocalHost.IPReservations[j].LocalHostID
			})
			sort.Slice(actualLocalHost.IPReservations, func(i, j int) bool {
				if actualLocalHost.IPReservations[i].Address != actualLocalHost.IPReservations[j].Address {
					return actualLocalHost.IPReservations[i].Address < actualLocalHost.IPReservations[j].Address
				}
				return actualLocalHost.IPReservations[i].LocalHostID < actualLocalHost.IPReservations[j].LocalHostID
			})

			for k := range expectedLocalHost.IPReservations {
				expectedReservation := expectedLocalHost.IPReservations[k]
				actualReservation := actualLocalHost.IPReservations[k]
				require.Equal(t, expectedReservation.Address, actualReservation.Address)
				require.Equal(t, expectedReservation.LocalHostID, actualReservation.LocalHostID)
			}
		}
	}
}

// Test that the 58 migration passes if the local_host table is not empty.
func TestMigrationFrom57(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add test data.
	_, _ = storktestdbmodel.AddTestHosts(t, db)

	expectedHosts, err := dbmodel.GetAllHosts(db, 0)
	require.NoError(t, err)

	// Act
	// Down to the previous migration.
	_, _, errDown := dbops.Migrate(db, "down", "57")
	// And back to the 56 migration.
	_, _, errUp := dbops.Migrate(db, "up")

	// Assert
	require.NoError(t, errDown)
	require.NoError(t, errUp)
	actualHosts, _ := dbmodel.GetAllHosts(db, 0)
	require.NotEmpty(t, expectedHosts)

	assertHostsTheSame(t, expectedHosts, actualHosts)
}

// Test that the 56 migration passes if the local_host table is not empty and
// the host details differ between the daemons.
// Some data are lost: the association between particular IP reservation and
// a daemon and the hostnames defined in the other local hosts than the first
// one.
func TestMigrationFrom57DifferentHostData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{Address: "cool.example.org", AgentPort: 8080}
	_ = dbmodel.AddMachine(db, m)

	subnet := &dbmodel.Subnet{
		ID:     1,
		Prefix: "192.0.2.0/24",
	}
	_ = dbmodel.AddSubnet(db, subnet)

	var daemons []*dbmodel.Daemon
	for i := 0; i < 2; i++ {
		accessPoints := []*dbmodel.AccessPoint{{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     int64(8080 + i),
			Key:      "",
			Protocol: protocoltype.HTTPS,
		}}

		daemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
		_ = daemon.SetKeaConfigFromJSON([]byte(`{
			"Dhcp4": {
				"client-classes": [
					{
						"name": "class2"
					},
					{
						"name": "class1"
					}
				],
				"subnet4": [
					{
						"id": 111,
						"subnet": "192.0.2.0/24"
					}
				],
				"hooks-libraries": [
					{
						"library": "libdhcp_host_cmds.so"
					}
				]
			}
		}`))
		err := dbmodel.AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
	}

	// Associate the daemons with the subnets.
	_ = dbmodel.AddDaemonToSubnet(db, subnet, daemons[0])
	_ = dbmodel.AddDaemonToSubnet(db, subnet, daemons[1])

	host := &dbmodel.Host{
		SubnetID: 1,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				DaemonID:       daemons[0].ID,
				Hostname:       "foo.example.org",
				DataSource:     dbmodel.HostDataSourceAPI,
				NextServer:     "192.2.2.1",
				ServerHostname: "foo.example.org",
				BootFileName:   "/tmp/foo.xyz",
				IPReservations: []dbmodel.IPReservation{
					{
						Address: "192.0.2.2",
					},
					{
						Address: "192.0.2.3",
					},
				},
			},
			{
				DaemonID:       daemons[1].ID,
				Hostname:       "bar.example.org",
				DataSource:     dbmodel.HostDataSourceAPI,
				NextServer:     "192.2.2.4",
				ServerHostname: "bar.example.org",
				BootFileName:   "/tmp/bar.xyz",
				IPReservations: []dbmodel.IPReservation{
					{
						Address: "192.0.2.5",
					},
					{
						Address: "192.0.2.6",
					},
				},
			},
		},
	}
	err := dbmodel.AddHost(db, host)
	require.NoError(t, err)

	initialHosts, _ := dbmodel.GetAllHosts(db, 0)

	// Act
	// Down to the previous migration.
	_, _, errDown := dbops.Migrate(db, "down", "57")
	// And back to the latest migration.
	_, _, errUp := dbops.Migrate(db, "up")

	// Assert
	require.NoError(t, errDown)
	require.NoError(t, errUp)
	actualHosts, _ := dbmodel.GetAllHosts(db, 0)
	require.NotEmpty(t, initialHosts)

	// The IP reservations has been merged.
	for i := 0; i < 2; i++ {
		require.Len(t,
			actualHosts[0].LocalHosts[i].IPReservations,
			len(initialHosts[0].LocalHosts[0].IPReservations)+
				len(initialHosts[0].LocalHosts[1].IPReservations),
		)
	}

	// Check the second local host has the same hostname as the first one.
	require.Equal(t,
		actualHosts[0].LocalHosts[0].Hostname,
		actualHosts[0].LocalHosts[1].Hostname,
	)
	require.Equal(t,
		initialHosts[0].LocalHosts[0].Hostname,
		actualHosts[0].LocalHosts[1].Hostname,
	)

	// Remove differences.
	for i := 0; i < 2; i++ {
		// Remove the IP reservations copied from another daemon.
		actualHosts[0].LocalHosts[i].IPReservations = initialHosts[0].LocalHosts[i].IPReservations[0:2]
	}
	// Restore the proper hostname of the second local host.
	actualHosts[0].LocalHosts[1].Hostname = initialHosts[0].LocalHosts[1].Hostname

	assertHostsTheSame(t, initialHosts, actualHosts)
}

// Test that the interval of the removed metrics puller is deleted.
func TestMigration59DeleteUnusedMetricsInterval(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)

	// Act
	_, _, errDown := dbops.Migrate(db, "down", "58")
	settingBefore, errBefore := dbmodel.GetSetting(db, "metrics_collector_interval")
	_, _, errUp := dbops.Migrate(db, "up", "59")
	settingAfter, errAfter := dbmodel.GetSetting(db, "metrics_collector_interval")

	// Assert
	require.NoError(t, errDown)
	require.NoError(t, errUp)
	require.NoError(t, errBefore)
	require.ErrorAs(t, errAfter, &pg.ErrNoRows)
	require.NotNil(t, settingBefore)
	require.Equal(t, "10", settingBefore.Value)
	require.Nil(t, settingAfter)
}

// Test that after migration the default user with a default password is
// forced to change the password.
func TestMigration60SetDefaultPasswordToChange(t *testing.T) {
	// Arrange & Act
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Assert
	user, err := dbmodel.GetUserByID(db, 1)
	require.NoError(t, err)
	require.True(t, user.ChangePassword)
	ok, err := dbmodel.Authenticate(db, user, "admin")
	require.NoError(t, err)
	require.True(t, ok)
}

// Test that the 55 migration passes if the local host is defined both in the
// database and the configuration file.
func TestMigration55LocalHostInDatabaseAndConfig(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbmodel.InitializeSettings(db, 0)

	// Prepare a configuration.
	server, _ := dbmodeltest.NewKeaDHCPv4Server(db)
	err := server.Configure(`{
		"Dhcp4": {
			"reservations": [
				{
					"ip-address": "192.0.2.204",
					"hostname": "foo.example.org"
				}
			]
		}
	}`)
	require.NoError(t, err)

	machine, err := server.GetMachine()
	require.NoError(t, err)
	fec := &storktestdbmodel.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	err = kea.CommitDaemonsIntoDB(db,
		machine.Daemons,
		fec,
		[]kea.DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
	require.NoError(t, err)

	// Add a database host reservations.
	hosts, _ := dbmodel.GetAllHosts(db, 0)
	for _, host := range hosts {
		var newLocalHosts []dbmodel.LocalHost
		for _, oldLocalHost := range host.LocalHosts {
			newLocalHost := oldLocalHost
			newLocalHost.ID = 0
			newLocalHost.DataSource = dbmodel.HostDataSourceAPI
			newLocalHosts = append(newLocalHosts, newLocalHost)
		}

		host.LocalHosts = append(host.LocalHosts, newLocalHosts...)
		err = dbmodel.UpdateHost(db, &host)
		require.NoError(t, err)
	}

	// Act
	_, _, err = dbops.Migrate(db, "reset")

	// Assert
	require.NoError(t, err)
}

// Test that it is possible to run all down migrations and reset the
// database schema when there are users with NULL email.
func TestDownMigration2NullUserEmail(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a user with NULL email.
	user := &dbmodel.SystemUser{
		Login:    "pablo",
		Lastname: "Valio",
		Name:     "Pablo",
	}
	_, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.NoError(t, err)

	// Make sure we can reset the database schema. Previously, the down migration
	// from version 2 to 1 would fail because it would try to apply a non NULL constraint
	// on NULL email column.
	_, newVersion, err := dbops.Migrate(db, "reset")
	require.NoError(t, err)
	require.Zero(t, newVersion)
}

// Test that database with some existing data can be migrated to the latest
// schema version.
// The dump file used in this test was created with Stork v2.3.0 using the
// demo. All machines except the one with many subnets were authorized.
// The dump was created immediately after the machine states were processed,
// and zones had been fetched without performing any other operations.
func TestMigrateFromDemoV2_3_0ToLatest(t *testing.T) {
	// Arrange & Act
	db, _, teardown := dbtest.SetupDatabaseTestCaseFromDump(t, "testdata/dump-demo-v2.3.0.sql")
	defer teardown()

	// Assert
	// Check if the migration was successfully executed.
	version, err := dbops.CurrentVersion(db)
	require.NoError(t, err)
	require.Equal(t, expectedSchemaVersion, version)

	// The data must be preserved after the migration.
	// Users.
	users, _, err := dbmodel.GetUsersByPage(db, 0, 10, nil, "", dbmodel.SortDirAsc)
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, "admin", users[0].Login)
	require.Equal(t, "internal", users[0].AuthenticationMethodID)
	require.Equal(t, "admin", users[1].Login)
	require.Equal(t, "ldap", users[1].AuthenticationMethodID)

	// Machines.
	machines, _, err := dbmodel.GetMachinesByPage(db, 0, 10, nil, nil, "", dbmodel.SortDirAny)
	require.NoError(t, err)
	require.Len(t, machines, 9)
	require.Equal(t, "agent-kea6", machines[0].Address)
	require.Equal(t, "agent-kea-ha3", machines[1].Address)
	require.Equal(t, "agent-kea-ha2", machines[2].Address)
	require.Equal(t, "agent-kea", machines[3].Address)
	require.Equal(t, "agent-kea-ha1", machines[4].Address)
	require.Equal(t, "agent-kea-large", machines[5].Address)
	require.Equal(t, "agent-pdns", machines[6].Address)
	require.Equal(t, "agent-bind9-2", machines[7].Address)
	require.Equal(t, "agent-bind9", machines[8].Address)

	// Daemons.
	daemons, err := dbmodel.GetAllDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 21)

	require.Len(t, machines[0].Daemons, 2)
	require.Equal(t, daemonname.CA, machines[0].Daemons[0].Name)
	require.True(t, machines[0].Daemons[0].Active)
	require.Len(t, machines[0].Daemons[0].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8000,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[0].Daemons[0].ID,
	}, *machines[0].Daemons[0].AccessPoints[0])
	require.Equal(t, daemonname.DHCPv6, machines[0].Daemons[1].Name)
	require.True(t, machines[0].Daemons[1].Active)
	require.Len(t, machines[0].Daemons[1].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8000,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[0].Daemons[1].ID,
	}, *machines[0].Daemons[1].AccessPoints[0])

	require.Len(t, machines[4].Daemons, 4)
	require.Equal(t, daemonname.DHCPv6, machines[4].Daemons[0].Name)
	require.False(t, machines[4].Daemons[0].Active)
	require.Len(t, machines[4].Daemons[0].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8001,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[4].Daemons[0].ID,
	}, *machines[4].Daemons[0].AccessPoints[0])
	require.Equal(t, daemonname.CA, machines[4].Daemons[1].Name)
	require.True(t, machines[4].Daemons[1].Active)
	require.Len(t, machines[4].Daemons[1].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8001,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[4].Daemons[1].ID,
	}, *machines[4].Daemons[1].AccessPoints[0])
	require.Equal(t, daemonname.D2, machines[4].Daemons[2].Name)
	require.False(t, machines[4].Daemons[2].Active)
	require.Len(t, machines[4].Daemons[2].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8001,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[4].Daemons[2].ID,
	}, *machines[4].Daemons[2].AccessPoints[0])
	require.Equal(t, daemonname.DHCPv4, machines[4].Daemons[3].Name)
	require.True(t, machines[4].Daemons[3].Active)
	require.Len(t, machines[4].Daemons[3].AccessPoints, 1)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     8001,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[4].Daemons[3].ID,
	}, *machines[4].Daemons[3].AccessPoints[0])

	require.Len(t, machines[8].Daemons, 1)
	require.Equal(t, daemonname.Bind9, machines[8].Daemons[0].Name)
	require.True(t, machines[8].Daemons[0].Active)
	require.Len(t, machines[8].Daemons[0].AccessPoints, 2)
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointStatistics,
		Address:  "127.0.0.1",
		Port:     8053,
		Protocol: protocoltype.HTTP,
		DaemonID: machines[8].Daemons[0].ID,
	}, *machines[8].Daemons[0].AccessPoints[0])
	require.Equal(t, dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "127.0.0.1",
		Port:     953,
		Protocol: protocoltype.HTTP,
		Key:      "rndc-key:hmac-sha256:C0WsVMnbpYt3RxJEZCrmJmlRyQJp9vy2lKp887r19mY=",
		DaemonID: machines[8].Daemons[0].ID,
	}, *machines[8].Daemons[0].AccessPoints[1])

	// Shared networks.
	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 4)

	require.Equal(t, "frog", sharedNetworks[0].Name)
	require.Equal(t, 4, sharedNetworks[0].Family)

	require.Equal(t, "mouse", sharedNetworks[1].Name)
	require.Equal(t, 4, sharedNetworks[1].Family)
	require.Len(t, sharedNetworks[1].LocalSharedNetworks, 1)
	require.Equal(t, machines[3].ID, sharedNetworks[1].LocalSharedNetworks[0].Daemon.MachineID)

	require.Equal(t, "frog", sharedNetworks[2].Name)
	require.Equal(t, 6, sharedNetworks[2].Family)

	require.Equal(t, "esperanto", sharedNetworks[3].Name)
	require.Equal(t, 4, sharedNetworks[3].Family)

	// Subnets.
	subnets, err := dbmodel.GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, subnets, 25)

	require.Equal(t, "192.1.15.0/24", subnets[7].Prefix)
	require.Equal(t, sharedNetworks[1].ID, subnets[7].SharedNetworkID)
	require.Len(t, subnets[7].LocalSubnets, 1)
	require.Equal(t, machines[3].ID, subnets[7].LocalSubnets[0].Daemon.MachineID)

	// Host reservations.
	hosts, err := dbmodel.GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, hosts, 37)

	host := hosts[24]
	require.Equal(t, 1, len(host.HostIdentifiers))
	identifier := host.HostIdentifiers[0]
	require.Equal(t, "hw-address", identifier.Type)
	require.Equal(t, []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, identifier.Value)
	require.Len(t, host.LocalHosts, 4)

	require.Equal(t, machines[4].Daemons[3].ID, host.LocalHosts[0].DaemonID)
	require.Equal(t, dbmodel.HostDataSourceConfig, host.LocalHosts[0].DataSource)
	require.Equal(t, "192.110.111.230/32", host.LocalHosts[0].IPReservations[0].Address)
	require.Equal(t, machines[1].Daemons[1].ID, host.LocalHosts[1].DaemonID)
	require.Equal(t, dbmodel.HostDataSourceAPI, host.LocalHosts[1].DataSource)
	require.Equal(t, "192.110.111.230/32", host.LocalHosts[1].IPReservations[0].Address)
	require.Equal(t, machines[2].Daemons[2].ID, host.LocalHosts[2].DaemonID)
	require.Equal(t, dbmodel.HostDataSourceAPI, host.LocalHosts[1].DataSource)
	require.Equal(t, "192.110.111.230/32", host.LocalHosts[1].IPReservations[0].Address)
	require.Equal(t, machines[4].Daemons[3].ID, host.LocalHosts[3].DaemonID)
	require.Equal(t, dbmodel.HostDataSourceAPI, host.LocalHosts[3].DataSource)
	require.Equal(t, "192.110.111.230/32", host.LocalHosts[3].IPReservations[0].Address)

	// HA status.
	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	require.Equal(t, "server3", services[0].HAService.Relationship)
	require.Equal(t, services[0].HAService.SecondaryID, services[0].Daemons[0].ID)
	require.Equal(t, machines[2].ID, services[0].Daemons[0].MachineID)
	require.Equal(t, services[0].HAService.PrimaryID, services[0].Daemons[1].ID)
	require.Equal(t, machines[1].ID, services[0].Daemons[1].MachineID)

	require.Equal(t, "server2", services[1].HAService.Relationship)
	require.Equal(t, services[1].HAService.PrimaryID, services[1].Daemons[0].ID)
	require.Equal(t, machines[4].ID, services[1].Daemons[0].MachineID)
	require.Equal(t, services[1].HAService.SecondaryID, services[1].Daemons[1].ID)
	require.Equal(t, machines[2].ID, services[1].Daemons[1].MachineID)

	// BIND zones.
	zones, _, err := dbmodel.GetZones(db, nil)
	require.NoError(t, err)
	require.Len(t, zones, 120)

	// Settings.
	settings, err := dbmodel.GetAllSettings(db)
	require.NoError(t, err)
	require.Len(t, settings, 10)

	expectSettings := map[string]any{
		"kea_status_puller_interval":      int64(30),
		"grafana_url":                     "",
		"grafana_dhcp4_dashboard_id":      "hRf18FvWz",
		"grafana_dhcp6_dashboard_id":      "AQPHKJUGz",
		"enable_machine_registration":     true,
		"state_puller_interval":           int64(30),
		"bind9_stats_puller_interval":     int64(60),
		"kea_stats_puller_interval":       int64(60),
		"kea_hosts_puller_interval":       int64(60),
		"enable_online_software_versions": true,
	}

	for expectedKey, expectedValue := range expectSettings {
		actualValue, ok := settings[expectedKey]
		require.True(t, ok, "setting %s not found", expectedKey)
		require.Equal(t, expectedValue, actualValue)
	}
}

// Test that the down migrations are executed properly for non-empty database.
func TestResetDatabaseWithData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseFromDump(t, "testdata/dump-demo-v2.3.0.sql")
	defer teardown()

	// Act
	_, _, err := dbops.Migrate(db, "down", "0")

	// Assert
	require.NoError(t, err)
}
