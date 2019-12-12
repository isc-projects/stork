package auth

import (
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/model"
	"net/http"
	"testing"
)

func TestAuthorize(t *testing.T) {
	user := &dbmodel.SystemUser{
		Id: 5,
		Groups: dbmodel.SystemGroups{
			&dbmodel.SystemGroup{
				Id:   2,
				Name: "admin",
			},
		},
	}

	req, _ := http.NewRequest("GET", "http://localhost:4200/api/users?start=0&limit=10", nil)
	ok, err := Authorize(user, req)
	require.NoError(t, err)
	require.False(t, ok)

	req, _ = http.NewRequest("GET", "http://example.com/users/5/", nil)
	ok, err = Authorize(user, req)
	require.NoError(t, err)
	require.True(t, ok)
}
