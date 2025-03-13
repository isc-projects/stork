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
	"isc.org/stork/server/apps/kea"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
)

// Current schema version. This value must be bumped up every
// time the schema is updated.
const expectedSchemaVersion int64 = 63

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

	_ = dbops.Toss(db)

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

	_ = dbops.Toss(db)

	// Migrate from version 0 to version 1.
	testMigrateAction(t, db, 0, 1, "up", "1")
}

// Tests that the database schema can be initialized and migrated to the
// latest version with one call.
func TestInitMigrateToLatest(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbops.Toss(db)

	o, n, err := dbops.MigrateToLatest(db)
	require.NoError(t, err)
	require.Zero(t, o)
	require.GreaterOrEqual(t, n, int64(18))
}

// Test that available schema version is returned as expected.
func TestAvailableVersion(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbops.Toss(db)

	_, _, err := dbops.Migrate(db, "init")
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

	_ = dbops.Toss(db)

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
func TestMigrationFrom57To58(t *testing.T) {
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
func TestMigrationFrom57To58DifferentHostData(t *testing.T) {
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

	var apps []*dbmodel.App
	for i := 0; i < 2; i++ {
		accessPoints := []*dbmodel.AccessPoint{}
		accessPoints = dbmodel.AppendAccessPoint(accessPoints,
			dbmodel.AccessPointControl, "localhost", "", int64(1234+i), true)

		app := &dbmodel.App{
			MachineID:    m.ID,
			Type:         dbmodel.AppTypeKea,
			Name:         fmt.Sprintf("kea-%d", i),
			Active:       true,
			AccessPoints: accessPoints,
			Daemons: []*dbmodel.Daemon{
				dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
			},
		}

		_ = app.Daemons[0].SetConfigFromJSON(`{
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
		}`)

		_, _ = dbmodel.AddApp(db, app)
		apps = append(apps, app)
	}

	// Associate the daemons with the subnets.
	_ = dbmodel.AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])

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
				DaemonID:       apps[0].Daemons[0].ID,
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
				DaemonID:       apps[1].Daemons[0].ID,
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
	// And back to the 58 migration.
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
	environment, _ := dbmodeltest.NewKea(db)
	server, _ := environment.NewKeaDHCPv4Server()
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

	fec := &storktestdbmodel.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	app, _ := server.GetKea()

	err = kea.CommitAppIntoDB(db, app, fec, nil, lookup)
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
