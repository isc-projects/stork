package dbmodel

import (
	"os"
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
)

var testConnOptions = dbops.PgOptions{
	Database: "storktest",
	User: "storktest",
	Password: "storktest",
}

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if _, ok := os.LookupEnv("POSTGRES_DB"); ok {
		testConnOptions.Addr = "postgres:5432"
	}

	// Toss the schema, including removal of the versioning table.
	dbmigs.Toss(&testConnOptions)
	dbmigs.Migrate(&testConnOptions, "init")
	dbmigs.Migrate(&testConnOptions, "up")

	// Run tests.
	c := m.Run()
	os.Exit(c)
}

func TestUserAddGet(t *testing.T) {
	db := pg.Connect(&testConnOptions)

    user1 := &SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
        Name:     "Jan",
		PasswordHash: "hash",
    }
    err := db.Insert(user1)

	require.NoError(t, err)

	user1Reflect := &SystemUser{Email: "xyz@info"}
	err = db.Model(user1Reflect).Where("email = ?", user1.Email).Select()
	require.NoError(t, err)

	require.Equal(t, user1.Lastname, user1Reflect.Lastname)
}
