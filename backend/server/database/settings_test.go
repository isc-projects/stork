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
func TestToConnectionStringNoSpaces(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123' host='localhost' port=123 sslmode='disable'", params)
}

// Test that the password including space character is enclosed in quotes.
func TestToConnectionStringWithSpaces(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' host='localhost' port=123 sslmode='disable'", params)
}

// Test that quotes and double quotes are escaped.
func TestToConnectionStringWithEscapes(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: `StOrK123'56"7`,
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ToConnectionString()
	require.Equal(t, `dbname='stork' user='admin' password='StOrK123\'56\"7' host='localhost' port=123 sslmode='disable'`, params)
}

// Test that when the host is not specified it is not included in the connection
// string.
func TestToConnectionStringWithOptionalHost(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Port:     123,
	}

	params := settings.ToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' port=123 sslmode='disable'", params)
}

// Test that when the port is 0, it is not included in the connection string.
func TestToConnectionStringWithOptionalPort(t *testing.T) {
	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		Host:     "localhost",
	}

	params := settings.ToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='stork' host='localhost' sslmode='disable'", params)
}

// Test that sslmode and related parameters are included in the connection string.
func TestToConnectionStringWithSSLMode(t *testing.T) {
	settings := DatabaseSettings{
		DBName:      "stork",
		User:        "admin",
		Password:    "stork",
		SSLMode:     "require",
		SSLCert:     "/tmp/sslcert",
		SSLKey:      "/tmp/sslkey",
		SSLRootCert: "/tmp/sslroot.crt",
	}

	params := settings.ToConnectionString()
	require.Equal(t, "dbname='stork' user='admin' password='stork' sslmode='require' sslcert='/tmp/sslcert' sslkey='/tmp/sslkey' sslrootcert='/tmp/sslroot.crt'", params)
}

// Test that toPgOptions function outputs SSL related parameters.
func TestToPgOptionsWithSSLMode(t *testing.T) {
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

	params, _ := settings.toPgOptions()
	require.NotNil(t, params)
	require.NotNil(t, params.TLSConfig)

	require.True(t, params.TLSConfig.InsecureSkipVerify)
	require.Nil(t, params.TLSConfig.VerifyConnection)
	require.Empty(t, params.TLSConfig.ServerName)
}

// Test that ToPgOptions function fails when there is an error in the
// SSL specific configuration.
func TestToPgOptionsWithWrongSSLModeSettings(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	settings := DatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		SSLMode:  "unsupported",
	}

	params, err := settings.toPgOptions()
	require.Nil(t, params)
	require.Error(t, err)
}

// Test that the TCP network kind is recognized properly.
func TestToPgOptionsTCP(t *testing.T) {
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
			options, err := settings.toPgOptions()

			// Assert
			require.NoError(t, err)
			require.EqualValues(t, "tcp", options.Network)
		})
	}
}

// Test that the socket is recognized properly.
func TestToPgOptionsSocket(t *testing.T) {
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
	options, err := settings.toPgOptions()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "unix", options.Network)
}
