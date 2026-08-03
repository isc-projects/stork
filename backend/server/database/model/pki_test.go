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
func TestGetSetSecret(t *testing.T) {
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

// Check if getting multiple secrets in database works.
func TestGetSecrets(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = SetSecret(db, "A", []byte("contentA"))
	_ = SetSecret(db, "B", []byte("contentB"))
	_ = SetSecret(db, "C", []byte("contentC"))

	t.Run("lexicographical order", func(t *testing.T) {
		// Act
		secrets, err := GetSecrets(db, "A", "B", "C")

		// Assert
		require.NoError(t, err)
		require.Equal(t, [][]byte{[]byte("contentA"), []byte("contentB"), []byte("contentC")}, secrets)
	})

	t.Run("random order", func(t *testing.T) {
		// Act
		secrets, err := GetSecrets(db, "C", "A", "B")

		// Assert
		require.NoError(t, err)
		require.Equal(t, [][]byte{[]byte("contentC"), []byte("contentA"), []byte("contentB")}, secrets)
	})

	t.Run("no keys", func(t *testing.T) {
		// Act
		secrets, err := GetSecrets(db)

		// Assert
		require.NoError(t, err)
		require.Nil(t, secrets)
	})

	t.Run("missing key", func(t *testing.T) {
		// Act
		secrets, err := GetSecrets(db, "")

		// Assert
		require.Error(t, err)
		require.Nil(t, secrets)
	})

	t.Run("unknown key", func(t *testing.T) {
		secrets, err := GetSecrets(db, "A", "B", "X", "C")

		// Assert
		require.Error(t, err)
		require.Nil(t, secrets)
	})
}
