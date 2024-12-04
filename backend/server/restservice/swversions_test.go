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
	secure := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(2, 0, 1),
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
	secures := []*ReportVersionDetails{
		&dev,
		&secure,
	}
	kea := ReportAppVersionMetadata{
		LatestDev:     &dev,
		CurrentStable: stables,
	}
	stork := ReportAppVersionMetadata{
		LatestDev:    &dev,
		LatestSecure: secures,
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

// Test that versionDetailsToRestAPI works fine.
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
	resultOne, err1 := versionDetailsToRestAPI(exampleOne)
	resultTwo, err2 := versionDetailsToRestAPI(exampleTwo)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, "1.2.3", *resultOne.Version)
	require.Equal(t, relDate, resultOne.ReleaseDate.String())
	require.Equal(t, int64(1), resultOne.Major)
	require.Equal(t, int64(2), resultOne.Minor)
	require.Equal(t, eolDate, resultOne.EolDate.String())
	require.Equal(t, esv, resultOne.Esv)
	require.Empty(t, resultOne.Range)
	require.Empty(t, resultOne.Status)

	require.Equal(t, "3.2.1", *resultTwo.Version)
	require.Equal(t, relDate, resultTwo.ReleaseDate.String())
	require.Equal(t, int64(3), resultTwo.Major)
	require.Equal(t, int64(2), resultTwo.Minor)
	require.Empty(t, resultTwo.EolDate)
	require.Empty(t, resultTwo.Esv)
	require.Empty(t, resultTwo.Range)
	require.Empty(t, resultTwo.Status)
}

// Test that versionDetailsToRestAPI returns error when date can't be parsed.
func TestVersionDetailsToRestAPIError(t *testing.T) {
	// Arrange & Act
	relDate := "2024-10-23"
	eolDate := "foobar"
	esv := "true"
	exampleOne := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(1, 2, 3),
		ReleaseDate: &relDate,
		EolDate:     eolDate,
		Esv:         esv,
	}
	resultOne, err1 := versionDetailsToRestAPI(exampleOne)

	relDate = "bad date"
	exampleTwo := ReportVersionDetails{
		Version:     storkutil.NewSemanticVersion(3, 2, 1),
		ReleaseDate: &relDate,
	}
	resultTwo, err2 := versionDetailsToRestAPI(exampleTwo)

	// Assert
	require.Error(t, err1)
	require.Error(t, err2)
	require.ErrorContains(t, err1, "failed to parse EoL date")
	require.ErrorContains(t, err2, "failed to parse release date")
	require.Nil(t, resultOne)
	require.Nil(t, resultTwo)
}

// Test that stableSwVersionsToRestAPI works fine.
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
	versionDetailsArr, stablesStringArr, err := stableSwVersionsToRestAPI(testData.Bind9.CurrentStable)

	// Assert
	require.NoError(t, err)
	for idx := range stablesStringArr {
		require.Equal(t, expectedStablesStrings[idx], stablesStringArr[idx])
		require.Equal(t, expectedRanges[idx], versionDetailsArr[idx].Range)
		require.Equal(t, "Current Stable", versionDetailsArr[idx].Status)
	}
}

// Test that appVersionMetadataToRestAPI works fine.
func TestAppVersionMetadataToRestAPI(t *testing.T) {
	// Arrange
	testData := getExampleData()

	// Act
	kea, err1 := appVersionMetadataToRestAPI(*testData.Kea)
	stork, err2 := appVersionMetadataToRestAPI(*testData.Stork)
	bind, err3 := appVersionMetadataToRestAPI(*testData.Bind9)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	require.Len(t, kea.CurrentStable, 2)
	require.Equal(t, "Current Stable", kea.CurrentStable[0].Status)
	require.Equal(t, "Current Stable", kea.CurrentStable[1].Status)
	require.Equal(t, "Development", kea.LatestDev.Status)
	require.Empty(t, kea.LatestSecure)
	require.Len(t, kea.SortedStableVersions, 2)
	require.Equal(t, "2.6.1", kea.SortedStableVersions[0])
	require.Equal(t, "9.20.2", kea.SortedStableVersions[1])

	require.Empty(t, stork.CurrentStable)
	require.Equal(t, "Development", stork.LatestDev.Status)
	require.Len(t, stork.LatestSecure, 2)
	require.Equal(t, "Security update", stork.LatestSecure[0].Status)
	require.Equal(t, "Security update", stork.LatestSecure[1].Status)

	require.Len(t, bind.CurrentStable, 2)
	require.Equal(t, "Current Stable", bind.CurrentStable[0].Status)
	require.Equal(t, "Current Stable", bind.CurrentStable[1].Status)
	require.Equal(t, "Development", bind.LatestDev.Status)
	require.Empty(t, bind.LatestSecure)
	require.Len(t, bind.SortedStableVersions, 2)
	require.Equal(t, "9.18.30", bind.SortedStableVersions[0])
	require.Equal(t, "9.20.2", bind.SortedStableVersions[1])
}

// Test that secureSoftwareVersionsToRestAPI works fine.
func TestSecureSoftwareVersionsToRestAPI(t *testing.T) {
	// Arrange
	testData := getExampleData()
	expectedRanges := []string{
		"2.7.x",
		"2.0.x",
	}

	// Act
	versionDetailsArr, err := secureSoftwareVersionsToRestAPI(testData.Stork.LatestSecure)

	// Assert
	require.NoError(t, err)
	for idx := range versionDetailsArr {
		require.Equal(t, expectedRanges[idx], versionDetailsArr[idx].Range)
		require.Equal(t, "Security update", versionDetailsArr[idx].Status)
	}
}
