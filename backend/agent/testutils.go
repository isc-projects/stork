package agent

import (
	"os"
	"path"

	"isc.org/stork/testutil"
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
// and generation error. This function always creates
// the files with the same content.
func GenerateSelfSignedCerts() (func(), error) {
	restoreCerts := RememberPaths()
	tmpDir, err := os.MkdirTemp("", "reg")
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
		restoreCerts()
	}

	err = os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	if err != nil {
		cleanup()
		return nil, err
	}
	err = os.Mkdir(path.Join(tmpDir, "tokens"), 0o755)
	if err != nil {
		cleanup()
		return nil, err
	}

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// store proper content
	err = os.WriteFile(KeyPEMFile, testutil.GetKeyPEMContent(), 0o600)
	if err != nil {
		cleanup()
		return nil, err
	}

	err = os.WriteFile(CertPEMFile, testutil.GetCertPEMContent(), 0o600)
	if err != nil {
		cleanup()
		return nil, err
	}

	err = os.WriteFile(RootCAFile, testutil.GetCACertPEMContent(), 0o600)
	if err != nil {
		cleanup()
		return nil, err
	}

	return cleanup, nil
}
