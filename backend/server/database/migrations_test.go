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
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
)

// Current schema version. This value must be bumped up every
// time the schema is updated.
const expectedSchemaVersion int64 = 58

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

// Test that the 56 migration passes if the local_host table is not empty.
func TestMigrationFrom55To56(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add test data.
	_, _ = storktestdbmodel.AddTestHosts(t, db)
	expectedHosts, err := dbmodel.GetAllHosts(db, 0)
	require.NoError(t, err)

	// Act
	// Down to the previous migration.
	_, _, errDown := dbops.Migrate(db, "down", "55")
	// And back to the 56 migration.
	_, _, errUp := dbops.Migrate(db, "up")

	// Assert
	require.NoError(t, errDown)
	require.NoError(t, errUp)
	actualHosts, _ := dbmodel.GetAllHosts(db, 0)
	require.NotEmpty(t, expectedHosts)
	require.Equal(t, len(expectedHosts), len(actualHosts))

	for i := range expectedHosts {
		expectedHost := expectedHosts[i]
		actualHost := actualHosts[i]
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
