package maintenance_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
)

// Constructs database username from the configured username and a custom
// prefix. It allows the Rake database tools to clean the users.
func prepareUsername(t *testing.T, suffix string) string {
	_, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	return fmt.Sprintf("%s_%s", settings.User, suffix)
}

// Test that the user role is created properly.
func TestCreateUser(t *testing.T) {
	// Arrange
	userName := prepareUsername(t, "user_create")
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	if ok, _ := maintenance.HasUser(db, userName); ok {
		_ = maintenance.DropUserIfExists(db, userName)
	}

	// Act
	err := maintenance.CreateUser(db, userName)
	defer func() {
		maintenance.DropUserIfExists(db, userName)
	}()

	// Assert
	require.NoError(t, err)
	ok, err := maintenance.HasUser(db, userName)
	require.NoError(t, err)
	require.True(t, ok)
}

// Test that the existence user check returns true if a user exists.
func TestHasUserForExistingUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_is_exists")
	if ok, _ := maintenance.HasUser(db, userName); !ok {
		_ = maintenance.CreateUser(db, userName)
	}

	// Act
	ok, err := maintenance.HasUser(db, userName)

	// Assert
	require.NoError(t, err)
	require.True(t, ok)
}

// Test that the existence user check returns false if a user doesn't exist.
func TestHasUserForNonExistingUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	ok, err := maintenance.HasUser(db, "stork_test_non_existing_user")

	// Assert
	require.NoError(t, err)
	require.False(t, ok)
}

// Test that dropping existing user works properly.
func TestDropUserIfExistsForExistingUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_drop_safe_existing")
	if ok, _ := maintenance.HasUser(db, userName); !ok {
		_ = maintenance.CreateUser(db, userName)
	}

	// Act
	err := maintenance.DropUserIfExists(db, userName)

	// Assert
	require.NoError(t, err)
}

// Test that dropping non-existing user causes no error.
func TestDropUserIfExistsForNonExistingUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_drop_safe_non_existing")
	if ok, _ := maintenance.HasUser(db, userName); ok {
		_ = maintenance.DropUserIfExists(db, userName)
	}

	// Act
	err := maintenance.DropUserIfExists(db, userName)

	// Assert
	require.NoError(t, err)
}

// Test that granting all privileges returns no error.
// In fact, this test doesn't check if the privileges were granted. We use a
// custom, newly created user, and it isn't allowed to log in, so we cannot
// check if it can perform restricted queries.
func TestGrantAllPrivileges(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_grant_all_privileges")
	if ok, _ := maintenance.HasUser(db, userName); !ok {
		_ = maintenance.CreateUser(db, userName)
	}

	// Act
	errDatabase := maintenance.GrantAllPrivilegesOnDatabaseToUser(db, settings.DBName, userName)
	errSchema := maintenance.GrantAllPrivilegesOnSchemaToUser(db, "public", userName)

	// Assert
	require.NoError(t, errDatabase)
	require.NoError(t, errSchema)
}

// Test that revoking all privileges returns no error.
// In fact, this test doesn't check if the privileges were revoked. We use a
// custom, newly created user, and it isn't allowed to log in, so we cannot
// check if it can perform restricted queries.
func TestRevokeAllPrivileges(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_revoke_all_privileges")
	if ok, _ := maintenance.HasUser(db, userName); !ok {
		_ = maintenance.CreateUser(db, userName)
	}
	_ = maintenance.GrantAllPrivilegesOnDatabaseToUser(db, settings.DBName, userName)
	_ = maintenance.GrantAllPrivilegesOnSchemaToUser(db, "public", userName)

	// Act
	err := maintenance.RevokeAllPrivilegesOnSchemaFromUser(db, "public", userName)
	// Revoke privileges for a non-existing user.
	errNotExists := maintenance.RevokeAllPrivilegesOnSchemaFromUser(db, "public", "stork_test_non_existing_user")

	// Assert
	require.NoError(t, err)
	require.Error(t, errNotExists)
}

// Test that the user password can be changed.
// In fact, this test doesn't check if the privileges were granted. We use a
// custom, newly created user, and it isn't allowed to log in, so we cannot
// check if it can perform restricted queries.
func TestAlterUserPassword(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	userName := prepareUsername(t, "user_alter_password")
	if ok, _ := maintenance.HasUser(db, userName); !ok {
		_ = maintenance.CreateUser(db, userName)
	}

	// Act
	err := maintenance.AlterUserPassword(db, userName, "foobar")

	// Assert
	require.NoError(t, err)
}

// Test that the password encryption is set and show properly.
func TestSetAndShowPasswordEncryption(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act & Assert
	err := maintenance.SetPasswordEncryption(db, maintenance.PgPasswordEncryptionMD5)
	require.NoError(t, err)
	passwordEncryption, err := maintenance.ShowPasswordEncryption(db)
	require.NoError(t, err)
	require.EqualValues(t, maintenance.PgPasswordEncryptionMD5, passwordEncryption)

	err = maintenance.SetPasswordEncryption(db, maintenance.PgPasswordEncryptionScramSHA256)
	require.NoError(t, err)
	passwordEncryption, err = maintenance.ShowPasswordEncryption(db)
	require.NoError(t, err)
	require.EqualValues(t, maintenance.PgPasswordEncryptionScramSHA256, passwordEncryption)
}

// Test that setting password encryption is temporarily and affects only the
// current connection.
func TestSetPasswordEncryptionAffectsOnlyCurrentConnection(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	initialPasswordEncryption, _ := maintenance.ShowPasswordEncryption(db)

	// Act
	// Change the password encryption to opposite.
	if initialPasswordEncryption == maintenance.PgPasswordEncryptionMD5 {
		_ = maintenance.SetPasswordEncryption(db, maintenance.PgPasswordEncryptionScramSHA256)
	} else {
		_ = maintenance.SetPasswordEncryption(db, maintenance.PgPasswordEncryptionMD5)
	}

	// Open a separate connection to the database.
	db, err := dbops.NewPgDBConn(settings)
	require.NoError(t, err)
	finishPasswordEncryption, err := maintenance.ShowPasswordEncryption(db)

	// Assert
	// The new connection shouldn't be affected by the password encryption change.
	require.NoError(t, err)
	require.EqualValues(t, initialPasswordEncryption, finishPasswordEncryption)
}
