package certs

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
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
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

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

// Check if ExportSecret correctly exports various keys and
// certificates from a database to a file.
func TestExportSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// setup server certs for the first time - they should be generated
	rootCertPEM, serverCertPEM, serverKeyPEM, err := SetupServerCerts(db)
	require.NoError(t, err)

	// check exporting Server Cert
	err = ExportSecret(db, dbmodel.SecretServerCert, "")
	require.NoError(t, err)

	serverCertPEMFile := sb.Join("server-cert.pem")
	err = ExportSecret(db, dbmodel.SecretServerCert, serverCertPEMFile)
	require.NoError(t, err)
	serverCertPEM2, err := ioutil.ReadFile(serverCertPEMFile)
	require.NoError(t, err)
	require.EqualValues(t, serverCertPEM, serverCertPEM2)

	// check exporting Server Key
	err = ExportSecret(db, dbmodel.SecretServerKey, "")
	require.NoError(t, err)

	serverKeyPEMFile := sb.Join("server-key.pem")
	err = ExportSecret(db, dbmodel.SecretServerKey, serverKeyPEMFile)
	require.NoError(t, err)
	serverKeyPEM2, err := ioutil.ReadFile(serverKeyPEMFile)
	require.NoError(t, err)
	require.EqualValues(t, serverKeyPEM, serverKeyPEM2)

	// check exporting CA Cert
	err = ExportSecret(db, dbmodel.SecretCACert, "")
	require.NoError(t, err)

	rootCertPEMFile := sb.Join("root-cert.pem")
	err = ExportSecret(db, dbmodel.SecretCACert, rootCertPEMFile)
	require.NoError(t, err)
	rootCertPEM2, err := ioutil.ReadFile(rootCertPEMFile)
	require.NoError(t, err)
	require.EqualValues(t, rootCertPEM, rootCertPEM2)

	// check exporting CA Key
	err = ExportSecret(db, dbmodel.SecretCAKey, "")
	require.NoError(t, err)

	rootKeyPEMFile := sb.Join("root-key.pem")
	err = ExportSecret(db, dbmodel.SecretCAKey, rootKeyPEMFile)
	require.NoError(t, err)
	rootKeyPEM2, err := ioutil.ReadFile(rootKeyPEMFile)
	require.NoError(t, err)
	rootKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCAKey)
	require.NoError(t, err)
	require.EqualValues(t, rootKeyPEM, rootKeyPEM2)

	// check exporting Server Token
	err = ExportSecret(db, dbmodel.SecretServerToken, "")
	require.NoError(t, err)

	serverTokenFile := sb.Join("server-token.txt")
	err = ExportSecret(db, dbmodel.SecretServerToken, serverTokenFile)
	require.NoError(t, err)
	serverToken2, err := ioutil.ReadFile(serverTokenFile)
	require.NoError(t, err)
	serverToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)
	require.EqualValues(t, serverToken, serverToken2)
}

// Check if ImportSecret correctly imports various keys and
// certificates from a file to a database.
func TestImportSecret(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// check importing Server Cert
	serverCertPEMFile := sb.Write("server-cert.pem", "abc")
	err := ImportSecret(db, dbmodel.SecretServerCert, serverCertPEMFile)
	require.NoError(t, err)
	serverCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerCert)
	require.NoError(t, err)
	require.EqualValues(t, serverCertPEM, "abc")

	// check importing Server Key
	serverKeyPEMFile := sb.Write("server-key.pem", "def")
	err = ImportSecret(db, dbmodel.SecretServerKey, serverKeyPEMFile)
	require.NoError(t, err)
	serverKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerKey)
	require.NoError(t, err)
	require.EqualValues(t, "def", serverKeyPEM)

	// check importing CA Cert
	rootCertPEMFile := sb.Write("root-cert.pem", "ghi")
	err = ImportSecret(db, dbmodel.SecretCACert, rootCertPEMFile)
	require.NoError(t, err)
	rootCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCACert)
	require.NoError(t, err)
	require.EqualValues(t, "ghi", rootCertPEM)

	// check importing CA Key
	rootKeyPEMFile := sb.Write("root-key.pem", "jkl")
	err = ImportSecret(db, dbmodel.SecretCAKey, rootKeyPEMFile)
	require.NoError(t, err)
	rootKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCAKey)
	require.NoError(t, err)
	require.EqualValues(t, "jkl", rootKeyPEM)

	// check importing Server Token
	serverTokenFile := sb.Write("server-token.txt", "mno")
	err = ImportSecret(db, dbmodel.SecretServerToken, serverTokenFile)
	require.NoError(t, err)
	serverToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)
	require.EqualValues(t, "mno", serverToken)
}
