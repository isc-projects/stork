package dumper

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/dumper/dumps"
)

// Test that the saver is properly constructed.
func TestConstructSaver(t *testing.T) {
	// Act
	saver := newTarbalSaver(
		json.Marshal,
		func(dump dumps.Dump, artifact dumps.Artifact) string { return "" },
	)

	// Assert
	require.NotNil(t, saver)
}

// Test that the saver creates the archive from the empty data.
func TestSaverSaveEmptyDumpList(t *testing.T) {
	// Arrange
	saver := newTarbalSaver(
		json.Marshal,
		func(dump dumps.Dump, artifact dumps.Artifact) string { return "" },
	)
	var buffer bytes.Buffer

	// Act
	err := saver.Save(&buffer, []dumps.Dump{})

	// Assert
	require.NoError(t, err)
	require.Len(t, buffer.Bytes(), 32)
}

// Test that the saver creates the archive from the non-empty data.
func TestSaverSaveFilledDumpList(t *testing.T) {
	// Arrange
	saver := newTarbalSaver(
		json.Marshal,
		func(dump dumps.Dump, artifact dumps.Artifact) string {
			return dump.Name() + artifact.Name()
		},
	)
	var buffer bytes.Buffer

	// Act
	dumps := []dumps.Dump{
		dumps.NewBasicDump(
			"foo",
			dumps.NewBasicStructArtifact("bar", 42),
		),
		dumps.NewBasicDump(
			"baz",
			dumps.NewBasicBinaryArtifact("biz", []byte{42, 24}),
			dumps.NewBasicStructArtifact("boz", "buz"),
		),
	}
	err := saver.Save(&buffer, dumps)

	// Assert
	require.NoError(t, err)
	require.Greater(t, buffer.Len(), 150)
}
