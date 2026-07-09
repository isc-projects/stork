package agent

import (
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Helper function that creates the temporary,
// self-signed certificates. Return the paths of the generated files, cleanup
// function and generation error.
func GenerateSelfSignedCerts() (paths certPaths, cleanup func(), err error) {
	sb := testutil.NewSandbox()

	cleanup = func() {
		sb.Close()
	}

	paths.certPath, paths.keyPath, paths.caPath, err = testutil.CreateTestCerts(sb)
	if err != nil {
		cleanup()
		return
	}

	token := [32]byte{24}
	tokenHex := storkutil.BytesToHex(token[:])
	paths.tokenPath, err = sb.Write("token.txt", tokenHex)
	if err != nil {
		cleanup()
		return
	}

	fingerprint := [32]byte{42}
	fingerprintHex := storkutil.BytesToHex(fingerprint[:])
	paths.fingerprintPath, err = sb.Write("server-cert.sha256", fingerprintHex)
	if err != nil {
		cleanup()
		return
	}

	return
}
