package main

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	goldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"isc.org/stork"
	"isc.org/stork/ldap"
	"isc.org/stork/server/certs"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Aux function checks if a list of expected strings is present in the string.
func checkOutput(output string, exp []string, reason string) bool {
	// The go-flags library wraps the output on certain width. We need to
	// remove the dashes and newlines to make the search work because the
	// os/exec Command function uses too narrow width.
	pattern := regexp.MustCompile(`-\n\n\s+`)
	output = pattern.ReplaceAllString(output, "")

	for _, x := range exp {
		if !strings.Contains(output, x) {
			fmt.Printf("ERROR: Expected string \"%s\" not found in %s.\n", x, reason)
			return false
		}
	}
	return true
}

// This is the list of all parameters we expect to be supported by stork-agent.
func getExpectedMainFragments() []string {
	return []string{
		"stork-tool",
		"-v",
		"--version",
		"-h",
		"--help",
		"cert-export",
		"cert-import",
		"db-init",
		"db-up",
		"db-down",
		"db-reset",
		"db-version",
		"db-set-version",
	}
}

// Location of the stork-agent binary.
const ToolBin = "./stork-tool"

//go:generate mockgen -package=main -destination=ldapdrivermock_test.go isc.org/stork/ldap LDAPDriver

// This test checks if all expected text fragments are documented in the man page.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open("../../../doc/user/man/stork-tool.8.rst")
	require.NoError(t, err)
	man, err := io.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedMainFragments(), "stork-tool.8.rst"))
}

// This test checks if stork-tool -h presents expected text fragments.
func TestMainHelp(t *testing.T) {
	// Run the --help version and get its output.
	toolCmd := exec.CommandContext(t.Context(), ToolBin, "-h")
	output, err := toolCmd.Output()
	require.NoError(t, err)

	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(output), getExpectedMainFragments(), "stork-tool -h output"))
}

// This test checks if stork-tool <cmd> -h commands present expected text fragments about db opts.
func TestDbOptsHelp(t *testing.T) {
	dbOpts := []string{
		"--db-url",
		"--db-user",
		"-u",
		"--db-password",
		"--db-host",
		"--db-port",
		"--db-sslmode",
		"--db-sslcert",
		"--db-sslkey",
		"--db-sslrootcert",
		"-p",
		"--db-name",
		"-d",
		"--db-trace-queries",
		"--db-tls-1-2-enabled",
		"-h",
		"--help",
		"STORK_DATABASE_",
	}

	cmds := []string{"db-init", "db-up", "db-down", "db-reset", "db-version", "db-set-version", "cert-export", "cert-import"}
	for _, cmd := range cmds {
		// Run the --help version and get its output.
		toolCmd := exec.CommandContext(t.Context(), ToolBin, cmd, "-h")
		output, err := toolCmd.Output()
		require.NoError(t, err)

		// Now check that all expected command-line switches are really there.
		require.True(t, checkOutput(
			string(output),
			dbOpts,
			fmt.Sprintf("stork-tool %s -h output", cmd),
		))
	}
}

// This test checks if stork-tool --version and -v report expected version.
func TestVersion(t *testing.T) {
	// #2482: clear the environment variables so that when we run
	// `rake unittest:backend_db`, the `STORK_DATABASE_IN_DOCKER` environment variable
	// doesn't make it into the test environment and cause the executable to warn about
	// an unknown `STORK_` environment variable.
	restore := testutil.ClearEnvironmentVariables()
	defer restore()
	// Let's repeat the test twice for -v and then for --version
	for _, opt := range []string{"-v", "--version"} {
		// Run the agent with specific switch.
		agentCmd := exec.CommandContext(t.Context(), ToolBin, opt)
		output, err := agentCmd.Output()
		require.NoError(t, err)

		// Clean up the output (remove end of line)
		ver := strings.TrimSpace(string(output))

		// Check if it equals expected version.
		require.Equal(t, stork.Version, ver)
	}
}

// This test checks if stork-tool --version and -v report expected version.
// It doesn't call the binary.
func TestVersionStandalone(t *testing.T) {
	// #2482: clear the environment variables so that when we run
	// `rake unittest:backend_db`, the `STORK_DATABASE_IN_DOCKER` environment variable
	// doesn't make it into the test environment and cause the executable to warn about
	// an unknown `STORK_` environment variable.
	restore := testutil.ClearEnvironmentVariables()
	defer restore()
	// Arrange
	app := newApp()

	for _, opt := range []string{"-v", "--version"} {
		t.Run(opt, func(t *testing.T) {
			args := []string{opt}

			// Act
			var err error
			stdout, _, captureErr := testutil.CaptureOutput(func() {
				err = app.Run("tool", args)
			})

			// Assert
			require.NoError(t, captureErr)
			require.NoError(t, err)
			require.Equal(t, stork.Version, strings.TrimSpace(string(stdout)))
		})
	}
}

// Check if a db-* command can be invoked.
func TestRunDBMigrate(t *testing.T) {
	_, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "db-up",
		"--db-name", settings.DBName,
		"--db-user", settings.User,
		"--db-password", settings.Password,
		"--db-host", settings.Host,
		"--db-port", strconv.Itoa(settings.Port),
	}
	main()
}

// Tests that migrating LDAP users updates external IDs only for LDAP users and
// keeps processing when one of the updates fails due to duplicated external ID.
func TestRunMigrateLDAPSystemUsers(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create 5 test users.
	// - internalUser: an internal user without LDAP authentication.
	// - ldapUserToUpdate: an LDAP user whose external ID will be updated.
	// - ldapUserToKeep: an LDAP user whose external ID should remain unchanged.
	// - ldapUserConflict: an LDAP user that will cause a conflict when
	//   updating external IDs.
	// - ldapUserToUpdateSecond: another LDAP user whose external ID will be
	//   updated to verify the migration is continued after the conflict error.
	internalUser := &dbmodel.SystemUser{
		Login:    "internal-user",
		Email:    "internal-user@example.org",
		Lastname: "Internal",
		Name:     "User",
	}
	_, err := dbmodel.CreateUser(db, internalUser)
	require.NoError(t, err)

	ldapUserToUpdate := &dbmodel.SystemUser{
		Login:                  "ldap-user-update",
		Email:                  "ldap-user-update@example.org",
		Lastname:               "LDAP",
		Name:                   "Update",
		AuthenticationMethodID: "ldap",
		ExternalID:             "ldap-old-id-1",
	}
	_, err = dbmodel.CreateUser(db, ldapUserToUpdate)
	require.NoError(t, err)

	ldapUserToKeep := &dbmodel.SystemUser{
		Login:                  "ldap-user-keep",
		Email:                  "ldap-user-keep@example.org",
		Lastname:               "LDAP",
		Name:                   "Keep",
		AuthenticationMethodID: "ldap",
		ExternalID:             "ldap-shared-id",
	}
	_, err = dbmodel.CreateUser(db, ldapUserToKeep)
	require.NoError(t, err)

	ldapUserConflict := &dbmodel.SystemUser{
		Login:                  "ldap-user-conflict",
		Email:                  "ldap-user-conflict@example.org",
		Lastname:               "LDAP",
		Name:                   "Conflict",
		AuthenticationMethodID: "ldap",
		ExternalID:             "ldap-old-id-2",
	}
	_, err = dbmodel.CreateUser(db, ldapUserConflict)
	require.NoError(t, err)

	ldapUserToUpdateSecond := &dbmodel.SystemUser{
		Login:                  "ldap-user-update-second",
		Email:                  "ldap-user-update-second@example.org",
		Lastname:               "LDAP",
		Name:                   "UpdateSecond",
		AuthenticationMethodID: "ldap",
		ExternalID:             "ldap-old-id-3",
	}
	_, err = dbmodel.CreateUser(db, ldapUserToUpdateSecond)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()

	ldapDriver := NewMockLDAPDriver(controller)
	ldapDriver.EXPECT().Dial("ldap://127.0.0.1:1389", gomock.Nil(), gomock.Any()).Return(nil)
	ldapDriver.EXPECT().SimpleBind("cn=bind,dc=example,dc=org", "bind-password", false).Return(nil)
	ldapDriver.EXPECT().Search(gomock.Any()).DoAndReturn(func(request *goldap.SearchRequest) (*goldap.SearchResult, error) {
		var uniqueID string
		switch {
		case strings.Contains(request.Filter, "(uid=ldap-user-update)"):
			uniqueID = "ldap-new-id"
		case strings.Contains(request.Filter, "(uid=ldap-user-keep)"):
			uniqueID = "ldap-shared-id"
		case strings.Contains(request.Filter, "(uid=ldap-user-conflict)"):
			uniqueID = "ldap-shared-id"
		case strings.Contains(request.Filter, "(uid=ldap-user-update-second)"):
			uniqueID = "ldap-new-id-2"
		default:
			return nil, fmt.Errorf("unexpected LDAP filter: %s", request.Filter)
		}

		return &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{Name: "entryUUID", Values: []string{uniqueID}},
					},
				},
			},
		}, nil
	}).Times(4)
	ldapDriver.EXPECT().Close()

	migrateSettings := &migrateLDAPSystemUsersSettings{
		DatabaseSettings: dbops.DatabaseCLIFlags{
			DBName:       settings.DBName,
			User:         settings.User,
			Password:     settings.Password,
			Host:         settings.Host,
			Port:         settings.Port,
			SSLMode:      settings.SSLMode,
			SSLCert:      settings.SSLCert,
			SSLKey:       settings.SSLKey,
			SSLRootCert:  settings.SSLRootCert,
			TLS12Enabled: settings.TLS12Enabled,
			TraceSQL:     "none",
			ReadTimeout:  settings.ReadTimeout,
			WriteTimeout: settings.WriteTimeout,
		},
		LDAPSettings: ldap.Settings{
			DialURL:      "ldap://127.0.0.1:1389",
			Root:         "dc=example,dc=org",
			BindUserDN:   "cn=bind,dc=example,dc=org",
			BindPassword: "bind-password",
			AttributeNames: ldap.LDAPAttributeNames{
				ObjectClassUser:  "organizationalPerson",
				UserID:           "uid",
				FirstName:        "givenName",
				LastName:         "sn",
				Email:            "mail",
				UniqueIdentifier: "entryUUID",
			},
		},
	}

	// Act
	err = runMigrateLDAPSystemUsers(migrateSettings, ldapDriver)

	// Assert
	require.NoError(t, err)

	// Untouched.
	internalUserAfter, err := dbmodel.GetUserByID(db, internalUser.ID)
	require.NoError(t, err)
	require.NotNil(t, internalUserAfter)
	require.Equal(t, dbmodel.AuthenticationMethodIDInternal, internalUserAfter.AuthenticationMethodID)
	require.Empty(t, internalUserAfter.ExternalID)

	// Updated. Got new external ID from LDAP.
	ldapUserToUpdateAfter, err := dbmodel.GetUserByID(db, ldapUserToUpdate.ID)
	require.NoError(t, err)
	require.NotNil(t, ldapUserToUpdateAfter)
	require.Equal(t, "ldap-new-id", ldapUserToUpdateAfter.ExternalID)

	// Updated. The external ID remains the same.
	ldapUserToKeepAfter, err := dbmodel.GetUserByID(db, ldapUserToKeep.ID)
	require.NoError(t, err)
	require.NotNil(t, ldapUserToKeepAfter)
	require.Equal(t, "ldap-shared-id", ldapUserToKeepAfter.ExternalID)

	// Updated. Got new external ID from LDAP.
	ldapUserToUpdateSecondAfter, err := dbmodel.GetUserByID(db, ldapUserToUpdateSecond.ID)
	require.NoError(t, err)
	require.NotNil(t, ldapUserToUpdateSecondAfter)
	require.Equal(t, "ldap-new-id-2", ldapUserToUpdateSecondAfter.ExternalID)

	// Untouched due to conflict.
	ldapUserConflictAfter, err := dbmodel.GetUserByID(db, ldapUserConflict.ID)
	require.NoError(t, err)
	require.NotNil(t, ldapUserConflictAfter)
	require.Equal(t, "ldap-old-id-2", ldapUserConflictAfter.ExternalID)
}

// Tests that LDAP migration processes more than one page of users.
func TestRunMigrateLDAPSystemUsersOver100(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, initialUserCount, err := dbmodel.GetUsersByPage(db, 0, 1, nil, "", "", dbmodel.SortDirAny)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()

	ldapDriver := NewMockLDAPDriver(controller)
	ldapDriver.EXPECT().Dial("ldap://127.0.0.1:1389", gomock.Nil(), gomock.Any()).Return(nil)
	ldapDriver.EXPECT().SimpleBind("cn=bind,dc=example,dc=org", "bind-password", false).Return(nil)

	const totalLDAPUsers = 101
	for i := 0; i < totalLDAPUsers; i++ {
		login := fmt.Sprintf("ldap-%03d", i)
		user := &dbmodel.SystemUser{
			Login:                  login,
			Email:                  fmt.Sprintf("%s@example.org", login),
			Lastname:               "Bulk",
			Name:                   "LDAP",
			AuthenticationMethodID: "ldap",
			ExternalID:             fmt.Sprintf("%s-old", login),
		}
		_, err := dbmodel.CreateUser(db, user)
		require.NoError(t, err)
	}

	uidFromFilterRegexp := regexp.MustCompile(`\(uid=([^\)]+)\)`)
	ldapDriver.EXPECT().Search(gomock.Any()).DoAndReturn(func(request *goldap.SearchRequest) (*goldap.SearchResult, error) {
		matches := uidFromFilterRegexp.FindStringSubmatch(request.Filter)
		if len(matches) != 2 {
			return nil, fmt.Errorf("unexpected LDAP filter: %s", request.Filter)
		}

		login := matches[1]
		uniqueID := fmt.Sprintf("%s-new", login)

		return &goldap.SearchResult{
			Entries: []*goldap.Entry{
				{
					Attributes: []*goldap.EntryAttribute{
						{Name: "entryUUID", Values: []string{uniqueID}},
					},
				},
			},
		}, nil
	}).Times(totalLDAPUsers)
	ldapDriver.EXPECT().Close()

	migrateSettings := &migrateLDAPSystemUsersSettings{
		DatabaseSettings: dbops.DatabaseCLIFlags{
			DBName:       settings.DBName,
			User:         settings.User,
			Password:     settings.Password,
			Host:         settings.Host,
			Port:         settings.Port,
			SSLMode:      settings.SSLMode,
			SSLCert:      settings.SSLCert,
			SSLKey:       settings.SSLKey,
			SSLRootCert:  settings.SSLRootCert,
			TLS12Enabled: settings.TLS12Enabled,
			TraceSQL:     "none",
			ReadTimeout:  settings.ReadTimeout,
			WriteTimeout: settings.WriteTimeout,
		},
		LDAPSettings: ldap.Settings{
			DialURL:      "ldap://127.0.0.1:1389",
			Root:         "dc=example,dc=org",
			BindUserDN:   "cn=bind,dc=example,dc=org",
			BindPassword: "bind-password",
			AttributeNames: ldap.LDAPAttributeNames{
				ObjectClassUser:  "organizationalPerson",
				UserID:           "uid",
				FirstName:        "givenName",
				LastName:         "sn",
				Email:            "mail",
				UniqueIdentifier: "entryUUID",
			},
		},
	}

	// Act
	err = runMigrateLDAPSystemUsers(migrateSettings, ldapDriver)

	// Assert
	require.NoError(t, err)

	// Verify a user expected to be handled on the second page was updated.
	secondPageLogin := "ldap-000"
	secondPageUser, err := dbmodel.GetUserByID(db, initialUserCount+1)
	require.NoError(t, err)
	require.NotNil(t, secondPageUser)
	require.Equal(t, fmt.Sprintf("%s-new", secondPageLogin), secondPageUser.ExternalID)

	// Verify a user from the first page was also updated.
	firstPageLogin := "ldap-100"
	firstPageUser, err := dbmodel.GetUserByID(db, initialUserCount+101)
	require.NoError(t, err)
	require.NotNil(t, firstPageUser)
	require.Equal(t, fmt.Sprintf("%s-new", firstPageLogin), firstPageUser.ExternalID)
}

// Check if cert-export can be invoked.
func TestRunCertExport(t *testing.T) {
	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, err := certs.GenerateServerToken(db)
	require.NoError(t, err)

	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "cert-export",
		"--db-name", settings.DBName,
		"--db-user", settings.User,
		"--db-password", settings.Password,
		"--db-host", settings.Host,
		"--db-port", strconv.Itoa(settings.Port),
		"-f", "srvtkn",
	}
	main()
}

// Check if cert-import can be invoked.
func TestRunCertImport(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, err := certs.GenerateServerToken(db)
	require.NoError(t, err)

	serverToken := "01234567890123456789001234567890" // 32-bytes
	require.EqualValues(t, len(serverToken), 32)
	srvTknFile, err := sb.Write("srv.tkn", serverToken)
	require.NoError(t, err)

	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "cert-import",
		"--db-name", settings.DBName,
		"--db-user", settings.User,
		"--db-password", settings.Password,
		"--db-host", settings.Host,
		"--db-port", strconv.Itoa(settings.Port),
		"-f", "srvtkn",
		"-i", srvTknFile,
	}
	main()
}

// Check if db-create command can be invoked.
func TestRunDBCreate(t *testing.T) {
	_, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Generate unique database name and use the same name for the user.
	dbName := fmt.Sprintf("storktest%d", rand.Int63())
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "db-create",
		"--db-maintenance-name", settings.DBName,
		"--db-maintenance-user", settings.User,
		"--db-maintenance-password", settings.Password,
		"--db-name", dbName,
		"--db-user", dbName,
		"--db-password", settings.Password,
		"--db-host", settings.Host,
		"--db-port", strconv.Itoa(settings.Port),
	}
	main()
}

// Check if db-password-gen command can be invoked.
func TestRunDBGenPassword(*testing.T) {
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "db-password-gen",
	}
	main()
}

// Test that the hook inspect command is running properly for directory path.
func TestRunHookInspectDirectory(*testing.T) {
	directory, _ := os.Getwd()
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "hook-inspect", "-p", directory,
	}

	main()
}

// Test that the hook inspect command is running properly for file path.
func TestRunHookInspectFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		// TODO: enable this test on macOS when the compiler issue is addressed.
		// See the possibly related ticket: https://github.com/golang/go/issues/33072.
		t.Skip(`Skipping the test consistently failing on macOS due to: "fatal error: runtime: no plugin module data"`)
	}
	file, _ := os.Executable()
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "hook-inspect", "-p", file,
	}

	main()
}

// Test that the custom welcome message can be deployed in the specific
// location and later undeployed.
func TestRunDeployStaticView(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create the input file.
	filepath, err := sb.Write("input.html", "<p>Welcome to Stork!</p>")
	require.NoError(t, err)

	// Create the output directory for the static content. It is relative
	// to the sandbox path.
	outFilepath, err := sb.JoinDir("assets/static-page-content")
	require.NoError(t, err)

	// Deploy the welcome message using the created input file and the
	// output directory. The output directory is set to sandbox location.
	defer testutil.CreateOsArgsRestorePoint()()
	os.Args = []string{
		"stork-tool", "deploy-login-page-welcome",
		"-i", filepath,
		"-d", sb.BasePath,
	}
	main()

	// Make sure that the welcome message was copied.
	_, err = os.Stat(path.Join(outFilepath, "login-screen-welcome.html"))
	require.NoError(t, err)

	// Make sure that the file has expected contents.
	data, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.EqualValues(t, "<p>Welcome to Stork!</p>", data)

	// Try to undeploy the welcome file.
	os.Args = []string{
		"stork-tool", "undeploy-login-page-welcome",
		"-d", sb.BasePath,
	}
	main()

	// It should no longer exist.
	_, err = os.Stat(path.Join(outFilepath, "login-screen-welcome.html"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}
