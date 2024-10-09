package restservice

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Helper function generating test data.
func getExampleData() ReportAppsVersions {
	relDate := "2024-11-23"
	eolDate := "2026-12-31"
	esv := "true"
	dev := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(2, 7, 3),
		ReleaseDate: &relDate,
	}
	stableEsv := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(9, 18, 30),
		ReleaseDate: &relDate,
		EolDate:     eolDate,
		Esv:         esv,
	}
	stable1 := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(9, 20, 2),
		ReleaseDate: &relDate,
		EolDate:     eolDate,
	}
	stable2 := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(2, 6, 1),
		ReleaseDate: &relDate,
		EolDate:     eolDate,
	}
	stables := []*ReportVersionDetails{
		&stable1,
		&stable2,
	}
	bindStables := []*ReportVersionDetails{
		&stable1,
		&stableEsv,
	}
	kea := ReportAppVersionMetadata{
		LatestDev:     &dev,
		CurrentStable: stables,
	}
	stork := ReportAppVersionMetadata{
		LatestDev:    &dev,
		LatestSecure: &dev,
	}
	bind := ReportAppVersionMetadata{
		LatestDev:     &dev,
		CurrentStable: bindStables,
	}
	dataDate := "2024-10-09"
	return ReportAppsVersions{
		Kea:   &kea,
		Stork: &stork,
		Bind9: &bind,
		Date:  &dataDate,
	}
}

// Test that VersionDetailsToRestAPI works fine.
func TestVersionDetailsToRestAPI(t *testing.T) {
	// Arrange
	relDate := "2024-11-23"
	eolDate := "2026-12-31"
	esv := "true"
	exampleOne := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(1, 2, 3),
		ReleaseDate: &relDate,
		EolDate:     eolDate,
		Esv:         esv,
	}
	exampleTwo := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(3, 2, 1),
		ReleaseDate: &relDate,
	}

	// Act
	resultOne := VersionDetailsToRestAPI(exampleOne)
	resultTwo := VersionDetailsToRestAPI(exampleTwo)

	// Assert
	require.Equal(t, "1.2.3", *resultOne.Version)
	require.Equal(t, relDate, *resultOne.ReleaseDate)
	require.Equal(t, int64(1), resultOne.Major)
	require.Equal(t, int64(2), resultOne.Minor)
	require.Equal(t, eolDate, resultOne.EolDate)
	require.Equal(t, esv, resultOne.Esv)
	require.Empty(t, resultOne.Range)
	require.Empty(t, resultOne.Status)

	require.Equal(t, "3.2.1", *resultTwo.Version)
	require.Equal(t, relDate, *resultTwo.ReleaseDate)
	require.Equal(t, int64(3), resultTwo.Major)
	require.Equal(t, int64(2), resultTwo.Minor)
	require.Empty(t, resultTwo.EolDate)
	require.Empty(t, resultTwo.Esv)
	require.Empty(t, resultTwo.Range)
	require.Empty(t, resultTwo.Status)
}

// Test that StableSwVersionsToRestAPI works fine.
func TestStableSwVersionsToRestAPI(t *testing.T) {
	// Arrange
	testData := getExampleData()
	expectedStablesStrings := []string{
		"9.18.30",
		"9.20.2",
	}
	expectedRanges := []string{
		"9.20.x",
		"9.18.x",
	}

	// Act
	versionDetailsArr, stablesStringArr := StableSwVersionsToRestAPI(testData.Bind9.CurrentStable)

	// Assert
	for idx := range stablesStringArr {
		require.Equal(t, expectedStablesStrings[idx], stablesStringArr[idx])
		require.Equal(t, expectedRanges[idx], versionDetailsArr[idx].Range)
		require.Equal(t, "Current Stable", versionDetailsArr[idx].Status)
	}
}

// Test that AppVersionMetadataToRestAPI works fine.
func TestAppVersionMetadataToRestAPI(t *testing.T) {
	// Arrange
	testData := getExampleData()

	// Act
	kea := AppVersionMetadataToRestAPI(*testData.Kea)
	stork := AppVersionMetadataToRestAPI(*testData.Stork)
	bind := AppVersionMetadataToRestAPI(*testData.Bind9)

	// Assert
	require.Len(t, kea.CurrentStable, 2)
	require.Equal(t, "Current Stable", kea.CurrentStable[0].Status)
	require.Equal(t, "Current Stable", kea.CurrentStable[1].Status)
	require.Equal(t, "Development", kea.LatestDev.Status)
	require.Empty(t, kea.LatestSecure)
	require.Len(t, kea.SortedStables, 2)
	require.Equal(t, "2.6.1", kea.SortedStables[0])
	require.Equal(t, "9.20.2", kea.SortedStables[1])

	require.Empty(t, stork.CurrentStable)
	require.Equal(t, "Development", stork.LatestDev.Status)
	require.Equal(t, "Security update", stork.LatestSecure.Status)

	require.Len(t, bind.CurrentStable, 2)
	require.Equal(t, "Current Stable", bind.CurrentStable[0].Status)
	require.Equal(t, "Current Stable", bind.CurrentStable[1].Status)
	require.Equal(t, "Development", bind.LatestDev.Status)
	require.Empty(t, bind.LatestSecure)
	require.Len(t, bind.SortedStables, 2)
	require.Equal(t, "9.18.30", bind.SortedStables[0])
	require.Equal(t, "9.20.2", bind.SortedStables[1])
}

// Check if TestGetPotentialVersionsJSONLocations returns paths.
func TestGetPotentialVersionsJSONLocations(t *testing.T) {
	// Arrange & Act
	paths := getPotentialVersionsJSONLocations()

	// Assert
	require.Greater(t, len(paths), 1)
}
