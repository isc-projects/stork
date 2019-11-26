package dbops

import (
	"os"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPgDB(t *testing.T) {
	addr := os.Getenv("POSTGRES_ADDR")

	var host, port string
	if addr != "" {
		host, port, _ = net.SplitHostPort(addr)
	} else {
		host = "localhost"
		port = "5432"
	}
	portInt, _ := strconv.Atoi(port)

	settings := DatabaseSettings{
		BaseDatabaseSettings: BaseDatabaseSettings{
			DbName: "storktest",
			User: "storktest",
			Host: host,
			Port: portInt,
		},
	}
	os.Setenv("STORK_DATABASE_PASSWORD", "storktest")
	defer os.Setenv("STORK_DATABASE_PASSWORD", "")

	db, err := NewPgDB(&settings)
	require.NotNil(t, db)
	require.NoError(t, err)
}
