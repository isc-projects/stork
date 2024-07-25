package dbops_test

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"testing"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Skips the test if the connection is not performed over UNIX socket.
func skipIfNonLocalConnection(t *testing.T, settings *dbops.DatabaseSettings) {
	if settings.Host != "" && !storkutil.IsSocket(settings.Host) {
		t.Skip("This test is available only if the database is connected over UNIX socket.")
	}
}

// Skips the test if the connection is performed over UNIX socket.
func skipIfLocalConnection(t *testing.T, settings *dbops.DatabaseSettings) {
	if settings.Host == "" || storkutil.IsSocket(settings.Host) {
		t.Skip("This test is available only if the database is connected over TCP/IP.")
	}
}

// Skips the test if the connection to the database running in Docker is over localhost.
//
// Warning: The helper doesn't work if the unit tests are not started by the
// Rakefile.
//
// The trust auth-related tests fail if the database is running in Docker
// and the connection is established over localhost gateway.
// In that case the connection is improperly recognized as local. The
// database host in our configuration is set to 127.0.0.1, but in practice, the
// operating system redirects the connection to the Docker container.
// In this case Stork treats the database as local, but the container sees the
// connection as remote (the Docker IP gateway address) and requires a
// password.
//
// Postgres in Docker is usually (and by default) configured to use a non-trust
// authentication method. The trust method is explicitly discouraged by the
// Docker image documentation.
//
// Conditions triggering the problem:
//   - Not providing a password
//   - Postgres in Docker container configured to use the trust authentication
//     method
//   - Stork configured to connect the database using the localhost address
//   - Stork must be running on the same machine where the containers are hosted
func skipIfDockerDatabaseAndLocalhost(t *testing.T, settings *dbops.DatabaseSettings) {
	if settings.Host != "localhost" && settings.Host != "127.0.0.1" && settings.Host != "::1" {
		// Non-localhost.
		return
	}

	if value, _ := os.LookupEnv("STORK_DATABASE_IN_DOCKER"); value == "true" {
		t.Skip("This test is not available because the database is running " +
			"in Docker and connection is established over localhost.")
	}
}

// Skips the test if the pg_hba.conf file has no entry for a specific
// user, connection type, and database or if the entry is set for different
// authentication method.
// It doesn't check the address and netmask.
func skipIfMissingUserEntryInPgHBAFile(t *testing.T, dbi dbops.DBI, settings *dbops.DatabaseSettings, userName string, expectedAuthMethod maintenance.PgAuthMethod) {
	// Read pg_hba rules.
	rules, err := maintenance.GetPgHBAConfiguration(dbi)
	if err != nil {
		t.Skip("Permissions are not sufficient to read the pg_hba rules.")
		return
	}

	// Detect connection type.
	connectionType := maintenance.PgConnectionLocal
	if settings.Host != "" && !storkutil.IsSocket(settings.Host) {
		connectionType = maintenance.PgConnectionHost
	}

	// Search for an user entry.
	for _, rule := range rules {
		// Check connection type.
		if rule.Type != connectionType {
			continue
		}

		// Check database.
		ok := false
		for _, ruleDatabase := range rule.Database {
			if ruleDatabase == settings.DBName || ruleDatabase == "all" {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}

		// Check username.
		for _, ruleUserName := range rule.UserName {
			if ruleUserName != userName && ruleUserName != "all" {
				continue
			}

			// Basic check for address.
			// We assume the rule can be configured to localhost or to valid
			// remote address.
			if connectionType == maintenance.PgConnectionHost {
				isLocalhostDatabase := false
				if settings.Host == "localhost" || settings.Host == "127.0.0.1" || settings.Host == "::1" {
					isLocalhostDatabase = true
				}
				isLocalhostRule := false
				if rule.Address == "127.0.0.1" || rule.Address == "::1" {
					isLocalhostRule = true
				}
				if isLocalhostDatabase != isLocalhostRule {
					continue
				}
			}

			// Rule exists.
			if rule.AuthMethod != expectedAuthMethod {
				// From https://www.postgresql.org/docs/current/auth-pg-hba-conf.html .
				// The first record with a matching connection type, client
				// address, requested database, and user name is used to
				// perform authentication. There is no “fall-through” or
				// “backup”: if one record is chosen and the authentication
				// fails, subsequent records are not considered. If no record
				// matches, access is denied.
				t.Skipf("The '%s' user for the '%s' connection has the "+
					"pg_hba.conf rule but for the '%s' authentication "+
					"method, expected: '%s'", userName, connectionType,
					rule.AuthMethod, expectedAuthMethod)
			}
			// User is allowed to use a given authentication method.
			log.
				WithFields(log.Fields{
					"type":       connectionType,
					"userName":   userName,
					"authMethod": expectedAuthMethod,
				}).
				Infof("Found compatible pg_hba.conf rule: %+v", rule)
			return
		}
	}

	// Missing rule for a given user with an expected authentication method.
	t.Skipf("The '%s' user with the '%s' auth method for the '%s' connection is "+
		"missing in the pg_hba.conf file",
		userName, expectedAuthMethod, connectionType)
}

// Tests database connection.
func checkDatabaseConnections(settings *dbops.DatabaseSettings) error {
	// Go-PG
	db, err := dbops.NewPgDBConn(settings)
	if err != nil {
		return errors.WithMessage(err, "cannot open database connection using go-pg")
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		return errors.Wrap(err, "cannot ping database using go-pg")
	}

	return nil
}

// Creates a custom database user dedicated to use with a specific
// authentication method. The username is prepared by concat the main testing
// database username (provided in DB configuration) and the authentication
// method; an underscore delimits these parts. Exceptions are peer and ident
// authentication methods that use the username of the current system user.
// Returns the username and teardown function responsible for deleting the user
// only if this function created it.
func setupCustomUser(t *testing.T, db *dbops.PgDB, authMethod maintenance.PgAuthMethod) (string, func()) {
	userTest := ""
	if authMethod == maintenance.PgAuthMethodPeer || authMethod == maintenance.PgAuthMethodIdentityServer {
		systemUser, _ := user.Current()
		userTest = systemUser.Username
	} else {
		_, settings, teardown := dbtest.SetupDatabaseTestCase(t)
		teardown()
		baseUser := settings.User
		userTest = fmt.Sprintf("%s_%s", baseUser, authMethod)
	}

	userInitiallyExists, _ := maintenance.HasUser(db, userTest)
	if !userInitiallyExists {
		_ = maintenance.CreateUser(db, userTest)
	}

	if authMethod == maintenance.PgAuthMethodScramSHA256 {
		_ = maintenance.SetPasswordEncryption(db, maintenance.PgPasswordEncryptionScramSHA256)
	}

	if authMethod == maintenance.PgAuthMethodMD5 || authMethod == maintenance.PgAuthMethodScramSHA256 {
		_ = maintenance.AlterUserPassword(db, userTest, userTest)
	}

	return userTest, func() {
		if !userInitiallyExists {
			_ = maintenance.DropUserIfExists(db, userTest)
		}
	}
}

// Test that the Stork can establish connection to the database using the
// trust authentication method.
func TestConnectUsingTrustAuth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	skipIfDockerDatabaseAndLocalhost(t, settings)

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodTrust)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodTrust)

	settings.User = userTest
	settings.Password = ""

	// Act
	err := checkDatabaseConnections(settings)

	// Assert
	require.NoError(t, err)
}

// Test that the Stork can establish connection to the database using the
// ident authentication method.
func TestConnectUsingIdentAuth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Ident authentication is available only for the host connections.
	skipIfLocalConnection(t, settings)

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodIdentityServer)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodIdentityServer)

	settings.User = userTest
	settings.Password = ""

	// Act
	err := checkDatabaseConnections(settings)

	// Assert
	require.NoError(t, err)
}

// Test that the Stork can establish connection to the database using the
// peer authentication method.
func TestConnectUsingPeerAuth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Peer authentication is available only for the local connections.
	skipIfNonLocalConnection(t, settings)

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodPeer)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodPeer)

	settings.User = userTest
	settings.Password = ""

	// Act
	err := checkDatabaseConnections(settings)

	// Assert
	require.NoError(t, err)
}

// Test that the Stork can establish connection to the database using the
// MD5 authentication method.
func TestConnectUsingMD5Auth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodMD5)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodMD5)

	settings.User = userTest
	settings.Password = userTest

	// Act
	err := checkDatabaseConnections(settings)

	// Assert
	require.NoError(t, err)
}

// Test that the Stork can establish connection to the database using the
// scram-sha256 authentication method.
func TestConnectUsingScramSHA256Auth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodScramSHA256)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodScramSHA256)

	settings.User = userTest
	settings.Password = userTest

	// Act
	err := checkDatabaseConnections(settings)

	// Assert
	require.NoError(t, err)
}
