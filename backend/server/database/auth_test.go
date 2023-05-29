package dbops_test

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Skips the test if the connection is not performed over UNIX socket.
func skipIfNonLocalConnection(t *testing.T, settings *dbops.DatabaseSettings) {
	if settings.Host != "" && !storkutil.IsSocket(settings.Host) {
		t.Skip("This test is available only if the database is local.")
	}
}

// Skips the test if the connection is performed over UNIX socket.
func skipIfLocalConnection(t *testing.T, settings *dbops.DatabaseSettings) {
	if settings.Host == "" || storkutil.IsSocket(settings.Host) {
		t.Skip("This test is available only if the database is not local.")
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
					"pg_hba.conf rule but for '%s authentication "+
					"method", userName, connectionType, rule.AuthMethod)
			}
			// User is allowed to use a given authentication method.
			return
		}
	}

	// Missing rule for a given user with an expected authentication method.
	t.Skipf("The '%s' user with the '%s' auth method for the '%s' connection is "+
		"missing in the pg_hba.conf file",
		userName, expectedAuthMethod, connectionType)
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

	return userTest, func() {
		if !userInitiallyExists {
			_ = maintenance.DropUserSafe(db, userTest)
		}
	}
}

// Test that the Stork can establish connection to the database using the
// trust authentication method.
func TestConnectUsingTrustAuth(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	userTest, teardownUser := setupCustomUser(t, db, maintenance.PgAuthMethodTrust)
	defer teardownUser()

	skipIfMissingUserEntryInPgHBAFile(t, db, settings, userTest, maintenance.PgAuthMethodTrust)

	settings.User = userTest
	settings.Password = ""

	// Act
	db, err := dbops.NewPgDBConn(settings)

	// Assert
	require.NoError(t, err)
	db.Close()
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
	db, err := dbops.NewPgDBConn(settings)

	// Assert
	require.NoError(t, err)
	db.Close()
}

// Test that the Stork can establish connection to the database using the
// ident authentication method.
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
	db, err := dbops.NewPgDBConn(settings)

	// Assert
	require.NoError(t, err)
	db.Close()
}
