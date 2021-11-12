package dumps_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/dumper/dumps"
)

// Test that the basic dump is constructed.
func TestBasicDump(t *testing.T) {
	// Act
	dump := dumps.NewBasicDump("foo")

	// Assert
	require.EqualValues(t, "foo", dump.Name())
	require.EqualValues(t, 0, dump.NumberOfArtifacts())
	require.NoError(t, dump.Execute())
}

// Test that the basic artifact is constructed.
func TestBasicArtifact(t *testing.T) {
	// Act
	artifact := dumps.NewBasicArtifact("foo")

	// Assert
	require.EqualValues(t, "foo", artifact.Name())
}

// Test that the basic dump with the artifacts is constructed.
func TestBasicDumpWithArtifacts(t *testing.T) {
	// Arrange
	first := dumps.NewBasicArtifact("bar")
	second := dumps.NewBasicArtifact("baz")

	// Act
	dump := dumps.NewBasicDump("foo", first, second)

	// Assert
	require.EqualValues(t, "foo", dump.Name())
	require.EqualValues(t, 2, dump.NumberOfArtifacts())
	require.NoError(t, dump.Execute())
	require.EqualValues(t, "bar", dump.GetArtifact(0).Name())
	require.EqualValues(t, "baz", dump.GetArtifact(1).Name())
}

// Test that the artifacts are appended.
func TestBasicDumpAppendArtifact(t *testing.T) {
	// Arrange
	dump := dumps.NewBasicDump("foo")

	// Act
	dump.AppendArtifact(dumps.NewBasicArtifact("bar"))
	dump.AppendArtifact(dumps.NewBasicArtifact("baz"))

	// Assert
	require.EqualValues(t, 2, dump.NumberOfArtifacts())
	require.EqualValues(t, "bar", dump.GetArtifact(0))
	require.EqualValues(t, "baz", dump.GetArtifact(1))
}

// Test that the basic struct artifact contains the data.
func TestBasicStructArtifact(t *testing.T) {
	// Arrange
	data := []string{"bar"}

	// Act
	artifact := dumps.NewBasicStructArtifact("foo", data)

	// Assert
	require.EqualValues(t, "foo", artifact.Name())
	require.Equal(t, data, artifact.GetStruct())
}

// Test that the basic struct artifact replaces the content.
func TestBasicStructArtifactSet(t *testing.T) {
	// Arrange
	data := []string{"bar"}
	artifact := dumps.NewBasicStructArtifact("foo", 42)

	// Act
	artifact.SetStruct(data)

	// Assert
	require.Equal(t, data, artifact.GetStruct())
}

// Test that the basic binary artifact is constructed.
func TestBasicBinaryArtifact(t *testing.T) {
	// Act
	artifact := dumps.NewBasicBinaryArtifact("foo", []byte("bar"))

	// Assert
	require.EqualValues(t, "foo", artifact.Name())
	require.EqualValues(t, []byte("bar"), artifact.GetBinary())
}
