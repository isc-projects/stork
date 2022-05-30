package dump_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	dumppkg "isc.org/stork/server/dumper/dump"
)

// Test that the basic dump is constructed.
func TestBasicDump(t *testing.T) {
	// Act
	dump := dumppkg.NewBasicDump("foo")

	// Assert
	require.EqualValues(t, "foo", dump.GetName())
	require.Zero(t, dump.GetArtifactsNumber())
	require.NoError(t, dump.Execute())
}

// Test that the basic artifact is constructed.
func TestBasicArtifact(t *testing.T) {
	// Act
	artifact := dumppkg.NewBasicArtifact("foo", ".ext")

	// Assert
	require.EqualValues(t, "foo", artifact.GetName())
	require.EqualValues(t, ".ext", artifact.GetExtension())
}

// Test that the basic dump with the artifacts is constructed.
func TestBasicDumpWithArtifacts(t *testing.T) {
	// Arrange
	first := dumppkg.NewBasicArtifact("bar", ".ear")
	second := dumppkg.NewBasicArtifact("baz", ".eaz")

	// Act
	dump := dumppkg.NewBasicDump("foo", first, second)

	// Assert
	require.EqualValues(t, "foo", dump.GetName())
	require.EqualValues(t, 2, dump.GetArtifactsNumber())
	require.NoError(t, dump.Execute())
	require.EqualValues(t, "bar", dump.GetArtifact(0).GetName())
	require.EqualValues(t, ".ear", dump.GetArtifact(0).GetExtension())
	require.EqualValues(t, "baz", dump.GetArtifact(1).GetName())
	require.EqualValues(t, ".eaz", dump.GetArtifact(1).GetExtension())
}

// Test that the artifacts are appended.
func TestBasicDumpAppendArtifact(t *testing.T) {
	// Arrange
	dump := dumppkg.NewBasicDump("foo")

	// Act
	dump.AppendArtifact(dumppkg.NewBasicArtifact("bar", ".ear"))
	dump.AppendArtifact(dumppkg.NewBasicArtifact("baz", ".eaz"))

	// Assert
	require.EqualValues(t, 2, dump.GetArtifactsNumber())
	require.EqualValues(t, "bar", dump.GetArtifact(0).GetName())
	require.EqualValues(t, ".ear", dump.GetArtifact(0).GetExtension())
	require.EqualValues(t, "baz", dump.GetArtifact(1).GetName())
	require.EqualValues(t, ".eaz", dump.GetArtifact(1).GetExtension())
}

// Test that the basic struct artifact contains the data.
func TestBasicStructArtifact(t *testing.T) {
	// Arrange
	data := []string{"bar"}

	// Act
	artifact := dumppkg.NewBasicStructArtifact("foo", data)

	// Assert
	require.EqualValues(t, "foo", artifact.GetName())
	require.Equal(t, data, artifact.GetStruct())
}

// Test that the basic struct artifact replaces the content.
func TestBasicStructArtifactSet(t *testing.T) {
	// Arrange
	data := []string{"bar"}
	artifact := dumppkg.NewBasicStructArtifact("foo", 42)

	// Act
	artifact.SetStruct(data)

	// Assert
	require.Equal(t, data, artifact.GetStruct())
}

// Test that the basic binary artifact is constructed.
func TestBasicBinaryArtifact(t *testing.T) {
	// Act
	artifact := dumppkg.NewBasicBinaryArtifact("foo", ".ext", []byte("bar"))

	// Assert
	require.EqualValues(t, "foo", artifact.GetName())
	require.EqualValues(t, ".ext", artifact.GetExtension())
	require.EqualValues(t, []byte("bar"), artifact.GetBinary())
}
