package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Check if generating serial numbers works.
func TestGetNewCertSerialNumber(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get first SN
	sn, err := GetNewCertSerialNumber(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, sn)

	// get second SN
	sn, err = GetNewCertSerialNumber(db)
	require.NoError(t, err)
	require.EqualValues(t, 2, sn)
}

// Check if getting and setting secrets in database works.
func TestGetSetSecrets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	keys := []string{
		SecretCAKey,
		SecretCACert,
		SecretServerKey,
		SecretServerCert,
		SecretServerToken,
	}

	for _, key := range keys {
		val, err := GetSecret(db, key)
		require.NoError(t, err)
		require.Nil(t, val)

		err = SetSecret(db, key, []byte("content"))
		require.NoError(t, err)

		val, err = GetSecret(db, key)
		require.NoError(t, err)
		require.EqualValues(t, "content", string(val))
	}
}
