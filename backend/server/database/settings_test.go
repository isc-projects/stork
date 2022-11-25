package dbops

import (
	"net"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that connection string is created when all parameters are specified and
// none of the values include a space character. Also, make sure that the password
// with upper case letters is handled correctly.
func TestConvertToConnectionStringNoSpaces(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123' host='localhost' port=123 sslmode='disable'", params)
}

// Test that the password including space character is enclosed in quotes.
func TestConvertToConnectionStringWithSpaces(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' host='localhost' port=123 sslmode='disable'", params)
}

// Test that quotes and double quotes are escaped.
func TestConvertToConnectionStringWithEscapes(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: `StOrK123'56"7`,
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, `dbname='stork' user='admin' password='StOrK123\'56\"7' host='localhost' port=123 sslmode='disable'`, params)
}

// Test that when the host is not specified it is not included in the connection
// string.
func TestConvertToConnectionStringWithOptionalHost(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Port:     123,
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' port=123 sslmode='disable'", params)
}

// Test that when the port is 0, it is not included in the connection string.
func TestConvertToConnectionStringWithOptionalPort(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		Host:     "localhost",
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='stork' host='localhost' sslmode='disable'", params)
}

// Test that sslmode and related parameters are included in the connection string.
func TestConvertToConnectionStringWithSSLMode(t *testing.T) {
	settings := DatabaseSettings{
		DBName:      "stork",
		User:        "admin",
		Password:    "stork",
		SSLMode:     "require",
		SSLCert:     "/tmp/sslcert",
		SSLKey:      "/tmp/sslkey",
		SSLRootCert: "/tmp/sslroot.crt",
	}

	params := settings.ConvertToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='stork' sslmode='require' sslcert='/tmp/sslcert' sslkey='/tmp/sslkey' sslrootcert='/tmp/sslroot.crt'", params)
}

// Test that convertToPgOptions function outputs SSL related parameters.
func TestConvertToPgOptionsWithSSLMode(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, _, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		SSLMode:  "require",
		SSLCert:  serverCert,
		SSLKey:   serverKey,
	}

	params, _ := settings.convertToPgOptions()
	require.NotNil(t, params)
	require.NotNil(t, params.TLSConfig)

	require.True(t, params.TLSConfig.InsecureSkipVerify)
	require.Nil(t, params.TLSConfig.VerifyConnection)
	require.Empty(t, params.TLSConfig.ServerName)
}

// Test that ConvertToPgOptions function fails when there is an error in the
// SSL specific configuration.
func TestConvertToPgOptionsWithWrongSSLModeSettings(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		SSLMode:  "unsupported",
	}

	params, err := settings.convertToPgOptions()
	require.Nil(t, params)
	require.Error(t, err)
}

// Test that the TCP network kind is recognized properly.
func TestConvertToPgOptionsTCP(t *testing.T) {
	// Arrange
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123",
		Port:     123,
	}

	hosts := []string{"localhost", "192.168.0.1", "fe80::42", "foo.bar"}

	for _, host := range hosts {
		settings.Host = host

		t.Run("host", func(t *testing.T) {
			// Act
			options, err := settings.convertToPgOptions()

			// Assert
			require.NoError(t, err)
			require.EqualValues(t, "tcp", options.Network)
		})
	}
}

// Test that the socket is recognized properly.
func TestConvertToPgOptionsSocket(t *testing.T) {
	// Arrange
	// Open a socket.
	socketDir := os.TempDir()
	socketPath := path.Join(socketDir, ".s.PGSQL.123")
	listener, _ := net.Listen("unix", socketPath)
	defer listener.Close()

	settings := DatabaseSettings{
		DBName:   "stork",
		Host:     socketDir,
		User:     "admin",
		Password: "StOrK123",
		Port:     123,
	}

	// Act
	options, err := settings.convertToPgOptions()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "unix", options.Network)
}

// Test that the field tags are handled properly.
func TestSetFieldsBasedOnTags(t *testing.T) {
	// Arrange
	type mock struct {
		FieldString              string `tag:"field-string"`
		FieldInt                 int    `tag:"field-int"`
		FieldWithoutTag          string
		FieldWithUnexpectedTag   string `unexpected:"tag"`
		FieldWithMultipleTags    string `tag:"field-multiple" another:"unexpected"`
		FieldWithUnsupportedType bool   `tag:"field-boolean"`
		FieldStringUnknown       string `tag:"field-unknown"`
	}

	lookup := func(key string) (string, bool) {
		switch key {
		case "field-string":
			return "value-string", true
		case "field-int":
			return "42", true
		case "field-multiple":
			return "value-multiple", true
		case "field-boolean":
			return "true", true
		default:
			return "", false
		}
	}

	obj := &mock{}

	// Act
	setFieldsBasedOnTags(obj, "tag", lookup)

	// Assert
	require.EqualValues(t, "value-string", obj.FieldString)
	require.EqualValues(t, 42, obj.FieldInt)
	require.Empty(t, obj.FieldWithoutTag)
	require.Empty(t, obj.FieldWithUnexpectedTag)
	require.EqualValues(t, "value-multiple", obj.FieldWithMultipleTags)
	require.False(t, obj.FieldWithUnsupportedType)
	require.Empty(t, obj.FieldStringUnknown)
}

// Test that the values of the struct members are read from environment
// variables correctly.
func TestReadFromEnvironment(t *testing.T) {
	// We need here a function to restore environment variables developed in #830.
	t.Fail()
}

type mockCLILookup struct {
	values map[string]string
}

func (m *mockCLILookup) IsSet(key string) bool {
	_, ok := m.values[key]
	return ok
}

func (m *mockCLILookup) String(key string) string {
	if value, ok := m.values[key]; ok {
		return value
	}
	return ""
}

// Test that the values of the struct members are read from CLI flags correctly.
func TestReadFromCLI(t *testing.T) {
	// Arrange
	type mock struct {
		FieldExisting string `long:"field-existing"`
		FieldMissing  string `long:"field-missing"`
	}

	lookup := &mockCLILookup{
		values: map[string]string{
			"field-existing": "value-existing",
		},
	}

	obj := &mock{}

	// Act
	readFromCLI(obj, lookup)

	// Assert
	require.EqualValues(t, "value-existing", obj.FieldExisting)
	require.Empty(t, obj.FieldMissing)
}

// Test that the database CLI flags are converted to the database settings
// properly.
func TestConvertDatabaseCLIFlagsToSettings(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlags{
		DBName:      "dbname",
		User:        "user",
		Password:    "password",
		Host:        "host",
		Port:        42,
		SSLMode:     "sslmode",
		SSLCert:     "sslcert",
		SSLKey:      "sslkey",
		SSLRootCert: "sslrootcert",
		TraceSQL:    "run",
	}

	// Act
	settings := cliFlags.ConvertToDatabaseSettings()

	// Assert
	require.EqualValues(t, "dbname", settings.DBName)
	require.EqualValues(t, "user", settings.User)
	require.EqualValues(t, "password", settings.Password)
	require.EqualValues(t, "host", settings.Host)
	require.EqualValues(t, 42, settings.Port)
	require.EqualValues(t, "sslmode", settings.SSLMode)
	require.EqualValues(t, "sslcert", settings.SSLCert)
	require.EqualValues(t, "sslkey", settings.SSLKey)
	require.EqualValues(t, "sslrootcert", settings.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, settings.TraceSQL)
}
