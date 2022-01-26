package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork"
	"isc.org/stork/server/certs"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Aux function checks if a list of expected strings is present in the string.
func checkOutput(output string, exp []string, reason string) bool {
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

// This test checks if all expected text fragments are documented in the man page.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open("../../../doc/man/stork-tool.8.rst")
	require.NoError(t, err)
	man, err := io.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedMainFragments(), "stork-tool.8.rst"))
}

// This test checks if stork-tool -h presents expected text fragments.
func TestMainHelp(t *testing.T) {
	// Run the --help version and get its output.
	toolCmd := exec.Command(ToolBin, "-h")
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
		"-h",
		"--help",
		"STORK_DATABASE_",
	}

	cmds := []string{"db-init", "db-up", "db-down", "db-reset", "db-version", "db-set-version", "cert-export", "cert-import"}
	for _, cmd := range cmds {
		// Run the --help version and get its output.
		toolCmd := exec.Command(ToolBin, cmd, "-h")
		output, err := toolCmd.Output()
		require.NoError(t, err)

		// Now check that all expected command-line switches are really there.
		require.True(t, checkOutput(string(output), dbOpts, "stork-tool * -h output"))
	}
}

// This test checks if stork-tool --version and -v report expected version.
func TestVersion(t *testing.T) {
	// Let's repeat the test twice for -v and then for --version
	for _, opt := range []string{"-v", "--version"} {
		// Run the agent with specific switch.
		agentCmd := exec.Command(ToolBin, opt)
		output, err := agentCmd.Output()
		require.NoError(t, err)

		// Clean up the output (remove end of line)
		ver := strings.TrimSpace(string(output))

		// Check if it equals expected version.
		require.Equal(t, ver, stork.Version)
	}
}

// Check if a db-* command can be invoked.
func TestRunDBMigrate(t *testing.T) {
	_, gOpts, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbOpts := gOpts.BaseDatabaseSettings

	os.Args = []string{
		"stork-tool", "db-up",
		"--db-name", dbOpts.DBName,
		"--db-user", dbOpts.User,
		"--db-password", dbOpts.Password,
		"--db-host", dbOpts.Host,
		"--db-port", strconv.Itoa(dbOpts.Port),
	}
	main()
}

// Check if cert-export can be invoked.
func TestRunCertExport(t *testing.T) {
	db, gOpts, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbOpts := gOpts.BaseDatabaseSettings

	_, err := certs.GenerateServerToken(db)
	require.NoError(t, err)

	os.Args = []string{
		"stork-tool", "cert-export",
		"--db-name", dbOpts.DBName,
		"--db-user", dbOpts.User,
		"--db-password", dbOpts.Password,
		"--db-host", dbOpts.Host,
		"--db-port", strconv.Itoa(dbOpts.Port),
		"-f", "srvtkn",
	}
	main()
}

// Check if cert-import can be invoked.
func TestRunCertImport(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	db, gOpts, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbOpts := gOpts.BaseDatabaseSettings

	_, err := certs.GenerateServerToken(db)
	require.NoError(t, err)

	serverToken := "01234567890123456789001234567890" // 32-bytes
	require.EqualValues(t, len(serverToken), 32)
	srvTknFile, err := sb.Write("srv.tkn", serverToken)
	require.NoError(t, err)

	os.Args = []string{
		"stork-tool", "cert-import",
		"--db-name", dbOpts.DBName,
		"--db-user", dbOpts.User,
		"--db-password", dbOpts.Password,
		"--db-host", dbOpts.Host,
		"--db-port", strconv.Itoa(dbOpts.Port),
		"-f", "srvtkn",
		"-i", srvTknFile,
	}
	main()
}

// Check if db-create command can be invoked.
func TestRunDBCreate(t *testing.T) {
	_, gOpts, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	dbOpts := gOpts.BaseDatabaseSettings

	// Generate unique database name and use the same name for the user.
	dbName := fmt.Sprintf("storktest%d", rand.Int63())
	os.Args = []string{
		"stork-tool", "db-create",
		"--db-admin-user", "storktest",
		"--db-admin-password", "storktest",
		"--db-name", dbName,
		"--db-user", dbName,
		"--db-password", dbOpts.Password,
		"--db-host", dbOpts.Host,
		"--db-port", strconv.Itoa(dbOpts.Port),
	}
	main()
}

// Check if db-password-gen command can be invoked.
func TestRunDBGenPassword(*testing.T) {
	os.Args = []string{
		"stork-tool", "db-password-gen",
	}
	main()
}
