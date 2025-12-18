package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiation of the detected daemon files.
func TestNewDetectedDaemonFiles(t *testing.T) {
	files := newDetectedDaemonFiles("/chroot", "/base")
	files.addFile(detectedFileTypeConfig, "config.conf")
	files.addFile(detectedFileTypeRndcKey, "rndc.key")
	require.Equal(t, "/chroot", files.chrootDir)
	require.Equal(t, "/base", files.baseDir)
	require.Len(t, files.files, 2)
	require.Equal(t, detectedFileTypeConfig, files.files[0].fileType)
	require.Equal(t, "config.conf", files.files[0].path)
	require.Equal(t, detectedFileTypeRndcKey, files.files[1].fileType)
	require.Equal(t, "rndc.key", files.files[1].path)
}

// Test getting the first file path by type.
func TestDetectedDaemonFilesGetFirstFilePathByType(t *testing.T) {
	files := newDetectedDaemonFiles("", "")
	files.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf")
	files.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key")
	path := files.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, "/etc/bind/config/config.conf", path)
	path = files.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Equal(t, "/etc/bind/rndc.key", path)
}
