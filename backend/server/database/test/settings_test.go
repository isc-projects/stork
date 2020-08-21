package dbtest

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbops "isc.org/stork/server/database"
)

// Test that connection string is created when all parameters are specified and
// none of the values include a space character. Also, make sure that the password
// with upper case letters is handled correctly.
func TestConnectionParamsNoQuotes(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DbName:   "stork",
		User:     "admin",
		Password: "StOrK123",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname=stork user=admin password=StOrK123 host=localhost port=123 sslmode=disable", params)
}

// Test that the password including space character is surrounded by quotes.
func TestConnectionParamsWithQuotes(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DbName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Host:     "localhost",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname=stork user=admin password='StOrK123 567' host=localhost port=123 sslmode=disable", params)
}

// Test that when the host is not specified it is not included in the connection
// string.
func TestConnectionParamsWithOptionalHost(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DbName:   "stork",
		User:     "admin",
		Password: "StOrK123 567",
		Port:     123,
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname=stork user=admin password='StOrK123 567' port=123 sslmode=disable", params)
}

// Test that when the port is 0, it is not included in the connection string.
func TestConnectionParamsWithOptionalPort(t *testing.T) {
	settings := dbops.BaseDatabaseSettings{
		DbName:   "stork",
		User:     "admin",
		Password: "stork",
		Host:     "localhost",
	}

	params := settings.ConnectionParams()
	require.Equal(t, "dbname=stork user=admin password=stork host=localhost sslmode=disable", params)
}
