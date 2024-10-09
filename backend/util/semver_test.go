package storkutil_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the semantic version is constructed correctly.
func TestNewSemanticVersion(t *testing.T) {
	// Arrange & Act
	semver := storkutil.NewSemanticVersion(1, 2, 3)

	// Assert
	require.Equal(t, 1, semver.Major)
	require.Equal(t, 2, semver.Minor)
	require.Equal(t, 3, semver.Patch)
}

// Test that the semantic version is converted to a string correctly.
func TestSemanticVersionString(t *testing.T) {
	// Arrange
	semver := storkutil.NewSemanticVersion(1, 2, 3)

	// Act
	str := semver.String()

	// Assert
	require.Equal(t, "1.2.3", str)
}

// Test that the semantic version is compared with a less semantic version correctly.
func TestSemanticVersionLessThan(t *testing.T) {
	// Arrange
	semver := storkutil.NewSemanticVersion(2, 2, 2)

	t.Run("less than", func(t *testing.T) {
		lessSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(0, 0, 0),
			storkutil.NewSemanticVersion(1, 2, 2),
			storkutil.NewSemanticVersion(2, 1, 2),
			storkutil.NewSemanticVersion(2, 2, 1),
		}

		for _, lessSemver := range lessSemVersions {
			// Act & Assert
			require.False(t, semver.LessThan(lessSemver))
			require.True(t, lessSemver.LessThan(semver))
		}
	})

	t.Run("equals", func(t *testing.T) {
		// Assert & Act
		require.False(t, semver.LessThan(semver))
	})

	t.Run("greater than", func(t *testing.T) {
		greaterSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(3, 2, 2),
			storkutil.NewSemanticVersion(2, 3, 2),
			storkutil.NewSemanticVersion(2, 2, 3),
		}

		for _, greaterSemver := range greaterSemVersions {
			// Act & Assert
			require.True(t, semver.LessThan(greaterSemver))
			require.False(t, greaterSemver.LessThan(semver))
		}
	})
}

// Test that the semantic version is compared with a greater semantic version correctly.
func TestSemanticVersionGreaterThan(t *testing.T) {
	// Arrange
	semver := storkutil.NewSemanticVersion(2, 2, 2)

	t.Run("less than", func(t *testing.T) {
		lessSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(0, 0, 0),
			storkutil.NewSemanticVersion(1, 2, 2),
			storkutil.NewSemanticVersion(2, 1, 2),
			storkutil.NewSemanticVersion(2, 2, 1),
		}

		for _, lessSemver := range lessSemVersions {
			// Act & Assert
			require.True(t, semver.GreaterThan(lessSemver))
			require.False(t, lessSemver.GreaterThan(semver))
		}
	})

	t.Run("equals", func(t *testing.T) {
		// Assert & Act
		require.False(t, semver.GreaterThan(semver))
	})

	t.Run("greater than", func(t *testing.T) {
		greaterSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(3, 2, 2),
			storkutil.NewSemanticVersion(2, 3, 2),
			storkutil.NewSemanticVersion(2, 2, 3),
		}

		for _, greaterSemver := range greaterSemVersions {
			// Act & Assert
			require.False(t, semver.GreaterThan(greaterSemver))
			require.True(t, greaterSemver.GreaterThan(semver))
		}
	})
}

// Test that the semantic version is compared with a less or equal semantic version correctly.
func TestSemanticVersionLessThanOrEqual(t *testing.T) {
	// Arrange
	semver := storkutil.NewSemanticVersion(2, 2, 2)

	t.Run("less than", func(t *testing.T) {
		lessSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(0, 0, 0),
			storkutil.NewSemanticVersion(1, 2, 2),
			storkutil.NewSemanticVersion(2, 1, 2),
			storkutil.NewSemanticVersion(2, 2, 1),
		}

		for _, lessSemver := range lessSemVersions {
			// Act & Assert
			require.False(t, semver.LessThanOrEqual(lessSemver))
			require.True(t, lessSemver.LessThanOrEqual(semver))
		}
	})

	t.Run("equals", func(t *testing.T) {
		// Assert & Act
		require.True(t, semver.LessThanOrEqual(semver))
	})

	t.Run("greater than", func(t *testing.T) {
		greaterSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(3, 2, 2),
			storkutil.NewSemanticVersion(2, 3, 2),
			storkutil.NewSemanticVersion(2, 2, 3),
		}

		for _, greaterSemver := range greaterSemVersions {
			// Act & Assert
			require.True(t, semver.LessThanOrEqual(greaterSemver))
			require.False(t, greaterSemver.LessThanOrEqual(semver))
		}
	})
}

// Test that the semantic version is compared with a greater or equal semantic version correctly.
func TestSemanticVersionGreaterThanOrEqual(t *testing.T) {
	// Arrange
	semver := storkutil.NewSemanticVersion(2, 2, 2)

	t.Run("less than", func(t *testing.T) {
		lessSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(0, 0, 0),
			storkutil.NewSemanticVersion(1, 2, 2),
			storkutil.NewSemanticVersion(2, 1, 2),
			storkutil.NewSemanticVersion(2, 2, 1),
		}

		for _, lessSemver := range lessSemVersions {
			// Act & Assert
			require.True(t, semver.GreaterThanOrEqual(lessSemver))
			require.False(t, lessSemver.GreaterThanOrEqual(semver))
		}
	})

	t.Run("equals", func(t *testing.T) {
		// Assert & Act
		require.True(t, semver.GreaterThanOrEqual(semver))
	})

	t.Run("greater than", func(t *testing.T) {
		greaterSemVersions := []storkutil.SemanticVersion{
			storkutil.NewSemanticVersion(3, 2, 2),
			storkutil.NewSemanticVersion(2, 3, 2),
			storkutil.NewSemanticVersion(2, 2, 3),
		}

		for _, greaterSemver := range greaterSemVersions {
			// Act & Assert
			require.False(t, semver.GreaterThanOrEqual(greaterSemver))
			require.True(t, greaterSemver.GreaterThanOrEqual(semver))
		}
	})
}

// Test that the semantic version is parsed correctly.
func TestParseSemanticVersion(t *testing.T) {
	// Arrange & Act
	semver, err := storkutil.ParseSemanticVersion("1.2.3")

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, semver.Major)
	require.Equal(t, 2, semver.Minor)
	require.Equal(t, 3, semver.Patch)
}

// Test that the invalid semantic version is not parsed.
func TestParseSemanticVersionError(t *testing.T) {
	// Arrange & Act
	semver, err := storkutil.ParseSemanticVersion("foobar")

	// Assert
	require.Error(t, err)
	require.Zero(t, semver.Major)
	require.Zero(t, semver.Minor)
	require.Zero(t, semver.Patch)
}

// Test that the invalid semantic version is parsed as the latest semantic version.
func TestParseSemanticVersionOrLatest(t *testing.T) {
	// Arrange & Act
	semver := storkutil.ParseSemanticVersionOrLatest("foobar")

	// Assert
	require.Equal(t, math.MaxInt, semver.Major)
	require.Equal(t, math.MaxInt, semver.Minor)
	require.Equal(t, math.MaxInt, semver.Patch)
}

// Test that SortSemversAsc works correctly.
func TestSortSemverAsc(t *testing.T) {
	// Arrange
	unsorted := []storkutil.SemanticVersion{
		storkutil.NewSemanticVersion(3, 2, 3),
		storkutil.NewSemanticVersion(1, 2, 3),
		storkutil.NewSemanticVersion(2, 3, 1),
		storkutil.NewSemanticVersion(1, 2, 3),
	}
	expected := []string{
		"1.2.3",
		"1.2.3",
		"2.3.1",
		"3.2.3",
	}

	// Act
	result := storkutil.SortSemversAsc(&unsorted)

	// Assert
	for idx := range result {
		require.Equal(t, expected[idx], result[idx])
	}
}

// Test that unmarshalling JSON with incorrect semver returns error.
func TestUnmarshalJSONError(t *testing.T) {
	// Arrange
	type TestJSON struct {
		Version storkutil.SemanticVersion
	}
	var unmarshalled TestJSON
	wrongJSON := `{
		"version": "foobar"
	}`

	// Act
	err := json.Unmarshal([]byte(wrongJSON), &unmarshalled)

	// Assert
	require.Error(t, err)
	require.Zero(t, unmarshalled.Version.Major)
	require.Zero(t, unmarshalled.Version.Minor)
	require.Zero(t, unmarshalled.Version.Patch)
}

// Test that unmarshalling JSON with correct semver works fine.
func TestUnmarshalJSON(t *testing.T) {
	// Arrange
	type TestJSON struct {
		Version storkutil.SemanticVersion
	}
	var unmarshalled TestJSON
	wrongJSON := `{
		"version": "1.2.3"
	}`

	// Act
	err := json.Unmarshal([]byte(wrongJSON), &unmarshalled)

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, unmarshalled.Version.Major)
	require.Equal(t, 2, unmarshalled.Version.Minor)
	require.Equal(t, 3, unmarshalled.Version.Patch)
}
