package agent

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the new instance of the log tailer can be created and that
// the internal fields have been initialized.
func TestNewLogTailer(t *testing.T) {
	lt := newLogTailer()
	require.NotNil(t, lt)
	require.NotNil(t, lt.allowedPaths)
	require.NotNil(t, lt.mutex)
}

// Test the mechanism which allows tailing selected files.
func TestAllow(t *testing.T) {
	lt := newLogTailer()
	lt.allow("/tmp/kea-dhcp4.log")
	require.True(t, lt.allowed("/tmp/kea-dhcp4.log"))
	require.False(t, lt.allowed("/tmp/kea-dhcp6.log"))

	// Make sure that it is ok to allow the same file twice.
	require.NotPanics(t, func() { lt.allow("/tmp/kea-dhcp4.log") })
	require.True(t, lt.allowed("/tmp/kea-dhcp4.log"))
}

// Test that if the file is not allowed an attempt to tail this file
// results in an error.
func TestTailForbidden(t *testing.T) {
	// Crate the test file to make sure that the lack of file is not
	// the reason for an error.
	filename := fmt.Sprintf("test%d.log", rand.Int63())
	f, err := os.Create(filename)
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(filename)
	}()
	fmt.Fprintln(f, "Some contents")

	// Tailing this file is initially not allowed, so an error should be returned.
	lt := newLogTailer()
	require.NotNil(t, lt)
	_, err = lt.tail(filename, 100)
	require.Error(t, err)

	// Allow tailing the file. This time there should be no error
	lt.allow(filename)
	_, err = lt.tail(filename, 100)
	require.NoError(t, err)
}

// Test that if the tailed file doesn't exist an error is returned.
func TestTailNotExistingFile(t *testing.T) {
	lt := newLogTailer()
	require.NotNil(t, lt)
	_, err := lt.tail("non-existing-file", 100)
	require.Error(t, err)
}
