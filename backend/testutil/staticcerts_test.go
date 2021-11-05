package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensure that these functions return something. It doesn't matter
// what because these functions are used in the unit tests which
// would fail if they weren't valid certs and keys.
func TestStaticCerts(t *testing.T) {
	require.NotEmpty(t, GetCACertPEMContent())
	require.NotEmpty(t, GetKeyPEMContent())
	require.NotEmpty(t, GetCertPEMContent())
}
