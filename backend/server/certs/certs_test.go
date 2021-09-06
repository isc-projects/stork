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

	const CAKEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg2hWn2DtPcXa2QL6F
19NZkPHMgnX+7FyyGuji4jiHKkahRANCAAQmZt58pZ/G92lKd3z41lXGM5sNbhtz
/UA27ZOOk/Z2xoZcUj7gl3Qce91mgcGE6DaLB1jqL9omNN+2loNAfAIE
-----END PRIVATE KEY-----`

	const CACERT = `-----BEGIN CERTIFICATE-----
MIIBjDCCATKgAwIBAgIBATAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTIxMDkwNjEyMjU0
N1oYDzIwNTEwOTA2MTIyNTQ3WjAzMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMRAwDgYDVQQDEwdSb290IENBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAEJmbefKWfxvdpSnd8+NZVxjObDW4bc/1ANu2TjpP2dsaGXFI+4Jd0HHvdZoHB
hOg2iwdY6i/aJjTftpaDQHwCBKM1MDMwEgYDVR0TAQH/BAgwBgEB/wIBATAdBgNV
HQ4EFgQUU07u+8zyLNobqvJi4rtpsSrayu8wCgYIKoZIzj0EAwIDSAAwRQIhAPAf
YfThoFyxzukrwN16eMP8lX8tVwhyNMZ0aRu3S4vdAiBAcDx0tFt+rWIyFz7eCkeB
fVkdWL4LIJypZP53JBCFYg==
-----END CERTIFICATE-----`

	const SRVKEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgwxFLpLqRpR46bS46
27ukTFCwOcL6I6NNEpfWSE8R+1yhRANCAAQMJcAWsP3nDDZdXYkeZI+D+IFozFbW
HJ/kNaPkCQjuBN2t02BZu6bdr2p5rXcK2mMbxvvjJhSXrBS0/jpsJKZs
-----END PRIVATE KEY-----`

	const SRVCERT = `-----BEGIN CERTIFICATE-----
MIICxDCCAmmgAwIBAgIBAjAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTIxMDkwNjEyMjU1
MFoYDzIwNTEwOTA2MTIyNTUwWjBGMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMQ8wDQYDVQQLEwZzZXJ2ZXIxEjAQBgNVBAMTCWxvY2FsaG9zdDBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAwlwBaw/ecMNl1diR5kj4P4gWjMVtYcn+Q1
o+QJCO4E3a3TYFm7pt2vanmtdwraYxvG++MmFJesFLT+OmwkpmyjggFXMIIBUzAf
BgNVHSMEGDAWgBRTTu77zPIs2huq8mLiu2mxKtrK7zCCAS4GA1UdEQSCASUwggEh
gglsb2NhbGhvc3SCBXR5Y2hvggV0eWNob4IFdHljaG+CDWlwNi1sb2NhbGhvc3SC
BXR5Y2hvggV0eWNob4IFdHljaG+CBXR5Y2hvggV0eWNob4IFdHljaG+CBXR5Y2hv
ggV0eWNob4cEfwAAAYcEwKgBY4cEwKh6AYcErBEAAYcQAAAAAAAAAAAAAAAAAAAA
AYcQIAEEcGOJAAAAAAAAAAAOLYcQ/Qq4KPD/AAAAAAAAAAAOLYcQ/Qq4KPD/AAD0
y69/jhT1zYcQ/Qq4KPD/AAAWRgqhR1EjJIcQIAEEcGOJAAAMjU3ezH+W/ocQIAEE
cGOJAABrU7hrjAdOgIcQ/oAAAAAAAADxjY42pn4t9IcQ/oAAAAAAAAAAQiX//obP
5DAKBggqhkjOPQQDAgNJADBGAiEAywycleZPDX5adSLRCghFA8476nVYmGlkwA7+
hbkkHg8CIQDEfP1HGySpXF5AhAK5RSIxSJTvVhzSSMKtAEmqG2BgYw==
-----END CERTIFICATE-----`

	const SRVTKN = "f3T93fi31WCNSFd7v62hm9f9lsJ7MoHt"

	// check importing Server Cert
	serverCertPEMFile, err := sb.Write("server-cert.pem", SRVCERT)
	require.NoError(t, err)
	err = ImportSecret(db, dbmodel.SecretServerCert, serverCertPEMFile)
	require.NoError(t, err)
	serverCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerCert)
	require.NoError(t, err)
	require.EqualValues(t, serverCertPEM, SRVCERT)

	// check importing Server Key
	serverKeyPEMFile, err := sb.Write("server-key.pem", SRVKEY)
	require.NoError(t, err)
	err = ImportSecret(db, dbmodel.SecretServerKey, serverKeyPEMFile)
	require.NoError(t, err)
	serverKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerKey)
	require.NoError(t, err)
	require.EqualValues(t, SRVKEY, serverKeyPEM)

	// check importing CA Cert
	rootCertPEMFile, err := sb.Write("root-cert.pem", CACERT)
	require.NoError(t, err)
	err = ImportSecret(db, dbmodel.SecretCACert, rootCertPEMFile)
	require.NoError(t, err)
	rootCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCACert)
	require.NoError(t, err)
	require.EqualValues(t, CACERT, rootCertPEM)

	// check importing CA Key
	rootKeyPEMFile, err := sb.Write("root-key.pem", CAKEY)
	require.NoError(t, err)
	err = ImportSecret(db, dbmodel.SecretCAKey, rootKeyPEMFile)
	require.NoError(t, err)
	rootKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCAKey)
	require.NoError(t, err)
	require.EqualValues(t, CAKEY, rootKeyPEM)

	// check importing Server Token
	serverTokenFile, err := sb.Write("server-token.txt", SRVTKN)
	require.NoError(t, err)
	err = ImportSecret(db, dbmodel.SecretServerToken, serverTokenFile)
	require.NoError(t, err)
	serverToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	require.NoError(t, err)
	require.EqualValues(t, SRVTKN, serverToken)

	// Check that importing non-existing file fails.
	err = ImportSecret(db, dbmodel.SecretServerCert, "nonexistent.txt")
	require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretServerKey, "nonexistent.txt")
	require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretCACert, "nonexistent.txt")
	require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretCAKey, "nonexistent.txt")
	require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretServerToken, "nonexistent.txt")
	require.Error(t, err)

	// Check that importing nonsense instead of valid PEM files is rejected.
	nonsenseFile, _ := sb.Write("nonsense.pem", "the Earth is flat")
	err = ImportSecret(db, dbmodel.SecretServerCert, nonsenseFile)
	require.Error(t, err)
	// Key validation is not implemented yet.
	// err = ImportSecret(db, dbmodel.SecretServerKey, nonsenseFile)
	// require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretCACert, nonsenseFile)
	require.Error(t, err)
	// Key validation is not implemented yet.
	// err = ImportSecret(db, dbmodel.SecretCAKey, nonsenseFile)
	// require.Error(t, err)
	err = ImportSecret(db, dbmodel.SecretServerToken, nonsenseFile)
	require.Error(t, err)
}
