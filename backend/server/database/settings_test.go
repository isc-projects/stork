package dbops

import (
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that convertToPgOptions function returns the default (empty) unix
// socket if the host is not provided.
func TestConvertToPgOptionsWithDefaultHost(t *testing.T) {
	// Arrange
	settings := DatabaseSettings{}

	// Act
	params, _ := settings.convertToPgOptions()

	// Assert
	require.Empty(t, params.Addr)
	require.EqualValues(t, "unix", params.Network)
}

// Test that convertToPgOptions function outputs SSL related parameters.
func TestConvertToPgOptionsWithSSLMode(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, _, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

	settings := DatabaseSettings{
		Host:     "http://postgres",
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
		Host:     "http://postgres",
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

// Test that the read and write timeouts are passed to the go-pg options.
func TestConvertToPgOptionsWithTimeouts(t *testing.T) {
	// Arrange
	settings := DatabaseSettings{
		DBName:       "stork",
		User:         "admin",
		Password:     "StOrK123",
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Hour,
	}

	// Act
	options, _ := settings.convertToPgOptions()

	// Assert
	require.EqualValues(t, 5*time.Minute, options.ReadTimeout)
	require.EqualValues(t, 10*time.Hour, options.WriteTimeout)
}

// Test that the string is converted into the logging query preset properly.
func TestNewLoggingQueryPreset(t *testing.T) {
	require.EqualValues(t, LoggingQueryPresetAll, newLoggingQueryPreset("all"))
	require.EqualValues(t, LoggingQueryPresetRuntime, newLoggingQueryPreset("run"))
	require.EqualValues(t, LoggingQueryPresetNone, newLoggingQueryPreset("none"))
	require.EqualValues(t, LoggingQueryPresetNone, newLoggingQueryPreset(""))
	require.EqualValues(t, LoggingQueryPresetNone, newLoggingQueryPreset("nil"))
	require.EqualValues(t, LoggingQueryPresetNone, newLoggingQueryPreset("false"))
}
