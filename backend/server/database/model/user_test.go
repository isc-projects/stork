package dbmodel

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
)

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Cleanup the database.
	dbtest.RecreateSchema()

	// Run tests.
	c := m.Run()
	os.Exit(c)
}

func TestUserAddGet(t *testing.T) {
	db := pg.Connect(&dbtest.PgConnOptions)

	fmt.Printf("%+v", dbtest.PgConnOptions)

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
