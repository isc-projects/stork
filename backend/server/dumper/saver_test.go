package dumper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/dumper/dump"
	storkutil "isc.org/stork/util"
)

// Test that the saver is properly constructed.
func TestConstructSaver(t *testing.T) {
	// Act
	saver := newTarballSaver(
		json.Marshal,
		func(dump dump.Dump, artifact dump.Artifact) string { return "" },
	)

	// Assert
	require.NotNil(t, saver)
}

// Test that the saver creates the archive from the empty data.
func TestSaverSaveEmptyDumpList(t *testing.T) {
	// Arrange
	saver := newTarballSaver(
		json.Marshal,
		func(dump dump.Dump, artifact dump.Artifact) string { return "" },
	)
	var buffer bytes.Buffer

	// Act
	err := saver.Save(&buffer, []dump.Dump{})

	// Assert
	require.NoError(t, err)
	// The empty tarball always has 32 bytes (using Go TAR and GZIP implementations).
	require.Len(t, buffer.Bytes(), 32)
}

// Test that the saver creates the archive from the non-empty data.
func TestSaverSaveFilledDumpList(t *testing.T) {
	// Arrange
	saver := newTarballSaver(
		json.Marshal,
		func(dump dump.Dump, artifact dump.Artifact) string {
			return dump.GetName() + artifact.GetName()
		},
	)
	var buffer bytes.Buffer

	// Act
	dumps := []dump.Dump{
		dump.NewBasicDump(
			"foo",
			dump.NewBasicStructArtifact("bar", 42),
		),
		dump.NewBasicDump(
			"baz",
			dump.NewBasicBinaryArtifact("biz", ".ext", []byte{42, 24}),
			dump.NewBasicStructArtifact("boz", "buz"),
		),
	}
	err := saver.Save(&buffer, dumps)

	// Assert
	require.NoError(t, err)
	require.GreaterOrEqual(t, buffer.Len(), 100)
}

// Test that the output tarball has proper content.
func TestSavedTarball(t *testing.T) {
	// Arrange
	saver := newTarballSaver(
		json.Marshal,
		func(dump dump.Dump, artifact dump.Artifact) string {
			return dump.GetName() + artifact.GetName()
		},
	)
	var buffer bytes.Buffer

	dumps := []dump.Dump{
		dump.NewBasicDump(
			"foo",
			dump.NewBasicStructArtifact("bar", 42),
		),
		dump.NewBasicDump(
			"baz",
			dump.NewBasicBinaryArtifact("biz", ".ext", []byte{42, 24}),
			dump.NewBasicStructArtifact("boz", "buz"),
		),
	}
	_ = saver.Save(&buffer, dumps)
	bufferBytes := buffer.Bytes()

	expectedFooBarContent, _ := json.Marshal(42)
	expectedBazBozContent, _ := json.Marshal("buz")

	// Act
	filenames, listErr := storkutil.ListFilesInTarball(bytes.NewReader(bufferBytes))
	fooBarContent, fooBarErr := storkutil.SearchFileInTarball(bytes.NewReader(bufferBytes), "foobar")
	bazBozContent, bazBozErr := storkutil.SearchFileInTarball(bytes.NewReader(bufferBytes), "bazboz")

	// Assert
	require.NoError(t, listErr)
	require.NoError(t, fooBarErr)
	require.NoError(t, bazBozErr)

	require.Len(t, filenames, 3)

	require.EqualValues(t, expectedFooBarContent, fooBarContent)
	require.EqualValues(t, expectedBazBozContent, bazBozContent)
}

// Test if the tarball is properly saved to file.
func TestSavedTarballToFile(t *testing.T) {
	// Arrange
	saver := newTarballSaver(
		json.Marshal,
		func(dump dump.Dump, artifact dump.Artifact) string {
			return dump.GetName() + artifact.GetName()
		},
	)
	file, _ := os.CreateTemp("", "*")
	defer (func() {
		file.Close()
		os.Remove(file.Name())
	})()

	bufferWriter := bufio.NewWriter(file)

	// Act
	err := saver.Save(bufferWriter, []dump.Dump{})
	_ = bufferWriter.Flush()
	stat, _ := file.Stat()
	position, _ := file.Seek(0, io.SeekCurrent)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, stat.Size())
	require.NotZero(t, position)
}
