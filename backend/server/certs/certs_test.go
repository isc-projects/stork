package certs

import (
	"testing"

	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"

	"github.com/stretchr/testify/require"
)

// Check if GenerateServerToken works.
func TestGenerateServerToken(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	token, err := GenerateServerToken(db)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	token2, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)
	require.EqualValues(t, token, token2)
}

// Check if SetupServerCerts works.
func TestSetupServerCerts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// setup server certs for the first time - they should be generated
	rootCertPEM, serverCertPEM, serverKeyPEM, err := SetupServerCerts(db)
	require.NoError(t, err)
	require.NotEmpty(t, rootCertPEM)
	require.NotEmpty(t, serverCertPEM)
	require.NotEmpty(t, serverKeyPEM)

	// setup server certs for the second time - they should be taken from database
	rootCertPEM2, serverCertPEM2, serverKeyPEM2, err := SetupServerCerts(db)
	require.NoError(t, err)
	require.EqualValues(t, rootCertPEM, rootCertPEM2)
	require.EqualValues(t, serverCertPEM, serverCertPEM2)
	require.EqualValues(t, serverKeyPEM, serverKeyPEM2)

	// destroy some certs and check if all is recreated
	secret := &dbmodel.Secret{}
	_, err = db.Model(secret).Where("name != ?", "").Delete()
	require.NoError(t, err)

	rootCertPEM3, serverCertPEM3, serverKeyPEM3, err := SetupServerCerts(db)
	require.NoError(t, err)
	require.NotEqualValues(t, rootCertPEM, rootCertPEM3)
	require.NotEqualValues(t, serverCertPEM, serverCertPEM3)
	require.NotEqualValues(t, serverKeyPEM, serverKeyPEM3)
}
