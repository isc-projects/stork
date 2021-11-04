package dbtest

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	testutil "isc.org/stork/testutil"
)

// Test that connection string is created when all parameters are specified and
// none of the values include a space character. Also, make sure that the password
// with upper case letters is handled correctly.
func TestConnectionParamsNoSpaces(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123' host='localhost' port=123 sslmode='disable'", params)
}

// Test that the password including space character is enclosed in quotes.
func TestConnectionParamsWithSpaces(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' host='localhost' port=123 sslmode='disable'", params)
}

// Test that quotes and double quotes are escaped.
func TestConnectionParamsWithEscapes(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: `StOrK123'56"7`,
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, `dbname='stork' user='admin' password='StOrK123\'56\"7' host='localhost' port=123 sslmode='disable'`, params)
}

// Test that when the host is not specified it is not included in the connection
// string.
func TestConnectionParamsWithOptionalHost(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname='stork' user='admin' password='StOrK123 567' port=123 sslmode='disable'", params)
}

// Test that when the port is 0, it is not included in the connection string.
func TestConnectionParamsWithOptionalPort(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:   "stork",
		User:     "admin",
		Password: "stork",
		Host:     "localhost",
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname='stork' user='admin' password='stork' host='localhost' sslmode='disable'", params)
}

// Test that sslmode and related parameters are included in the connection string.
func TestConnectionParamsWithSSLMode(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DBName:      "stork",
		User:        "admin",
		Password:    "stork",
		SSLMode:     "require",
		SSLCert:     "/tmp/sslcert",
		SSLKey:      "/tmp/sslkey",
		SSLRootCert: "/tmp/sslroot.crt",
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname='stork' user='admin' password='stork' sslmode='require' sslcert='/tmp/sslcert' sslkey='/tmp/sslkey' sslrootcert='/tmp/sslroot.crt'", params)
}

func TestPgParamsWithSSLMode(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, _ := createTestCerts(t, sb)

	settings := dbops.DatabaseSettings{
		BaseDatabaseSettings: dbops.BaseDatabaseSettings{
			DBName:   "stork",
			User:     "admin",
			Password: "stork",
			SSLMode:  "require",
			SSLCert:  serverCert,
			SSLKey:   serverKey,
		},
	}

	params, _ := settings.PgParams()
	require.NotNil(t, params)
	require.NotNil(t, params.TLSConfig)

	require.True(t, params.TLSConfig.InsecureSkipVerify)
	require.Nil(t, params.TLSConfig.VerifyConnection)
	require.Empty(t, params.TLSConfig.ServerName)
}

func TestPgParamsWithWrongSSLModeSettings(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	settings := dbops.DatabaseSettings{
		BaseDatabaseSettings: dbops.BaseDatabaseSettings{
			DBName:   "stork",
			User:     "admin",
			Password: "stork",
			SSLMode:  "unspported",
		},
	}

	params, err := settings.PgParams()
	require.Nil(t, params)
	require.Error(t, err)
}
