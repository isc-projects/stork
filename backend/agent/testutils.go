package agent

import (
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Helper function to store and defer restore
// original paths of: certificates, secrets and credentials.
func RememberPaths() func() {
	originalKeyPEMFile := KeyPEMFile
	originalCertPEMFile := CertPEMFile
	originalRootCAFile := RootCAFile
	originalAgentTokenFile := AgentTokenFile
	originalCredentialsFile := CredentialsFile

	return func() {
		KeyPEMFile = originalKeyPEMFile
		CertPEMFile = originalCertPEMFile
		RootCAFile = originalRootCAFile
		AgentTokenFile = originalAgentTokenFile
		CredentialsFile = originalCredentialsFile
	}
}

// Helper function that creates the temporary,
// self-signed certificates. Return the cleanup function
// and generation error.
func GenerateSelfSignedCerts() (func(), error) {
	restoreCerts := RememberPaths()
	sb := testutil.NewSandbox()

	cleanup := func() {
		sb.Close()
		restoreCerts()
	}

	cert, key, ca, err := testutil.CreateTestCerts(sb)
	if err != nil {
		cleanup()
		return nil, err
	}

	KeyPEMFile = key
	CertPEMFile = cert
	RootCAFile = ca

	var fingerprint [32]byte
	token := storkutil.BytesToHex(fingerprint[:])
	AgentTokenFile, err = sb.Write("token.txt", token)
	if err != nil {
		return nil, err
	}

	return cleanup, nil
}
