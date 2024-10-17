package restservice

import (
	"fmt"

	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// Structs used to deserialize offline versions.json report.
type ReportVersionDetails struct {
	EolDate     string                    `json:"eolDate,omitempty"`
	Esv         string                    `json:"esv,omitempty"`
	ReleaseDate *string                   `json:"releaseDate"`
	Version     storkutil.SemanticVersion `json:"version"`
}

type ReportAppVersionMetadata struct {
	CurrentStable []*ReportVersionDetails `json:"currentStable,omitempty"`
	LatestDev     *ReportVersionDetails   `json:"latestDev,omitempty"`
	LatestSecure  *ReportVersionDetails   `json:"latestSecure,omitempty"`
}

type ReportAppsVersions struct {
	Bind9 *ReportAppVersionMetadata `json:"bind9"`
	Date  *string                   `json:"date"`
	Kea   *ReportAppVersionMetadata `json:"kea"`
	Stork *ReportAppVersionMetadata `json:"stork"`
}

// VersionsJSON is a path to a JSON file with offline software versions metadata.
// It needs to be modified by tests so it has to be global variable.
var VersionsJSON = "/etc/stork/versions.json" //nolint:gochecknoglobals

// Get potential locations of versions.json.
func getPotentialVersionsJSONLocations() []string {
	return []string{
		VersionsJSON,
		"etc/versions.json",
	}
}

// Post processes either Kea, Bind9 or Stork version metadata and returns the data in REST API format.
func AppVersionMetadataToRestAPI(input ReportAppVersionMetadata) *models.AppVersionMetadata {
	out := models.AppVersionMetadata{}
	if input.LatestSecure != nil {
		out.LatestSecure = VersionDetailsToRestAPI(*input.LatestSecure)
		out.LatestSecure.Status = "Security update"
	}
	if input.LatestDev != nil {
		out.LatestDev = VersionDetailsToRestAPI(*input.LatestDev)
		out.LatestDev.Status = "Development"
	}
	if input.CurrentStable != nil {
		out.CurrentStable, out.SortedStableVersions = StableSwVersionsToRestAPI(input.CurrentStable)
	}
	return &out
}

// Post processes either Kea, Bind9 or Stork software release details and returns the data in REST API format.
func VersionDetailsToRestAPI(input ReportVersionDetails) *models.VersionDetails {
	v := input.Version.String()
	out := models.VersionDetails{
		Version:     &v,
		ReleaseDate: input.ReleaseDate,
		Major:       int64(input.Version.Major),
		Minor:       int64(input.Version.Minor),
	}
	if len(input.EolDate) > 0 {
		out.EolDate = input.EolDate
	}
	if len(input.Esv) > 0 {
		out.Esv = input.Esv
	}
	return &out
}

// Post processes either Kea, Bind9 or Stork stable release details and returns the data in REST API format.
// Takes an array of pointers to ReportVersionDetails for stable realeases.
// Returns an array of pointers to VersionDetails for stable realeases in REST API format
// and an array of strings with stable release semvers sorted in ascending order.
func StableSwVersionsToRestAPI(input []*ReportVersionDetails) ([]*models.VersionDetails, []string) {
	versionDetailsArr := []*models.VersionDetails{}
	stablesArr := []storkutil.SemanticVersion{}

	for _, details := range input {
		element := VersionDetailsToRestAPI(*details)
		stablesArr = append(stablesArr, details.Version)
		element.Status = "Current Stable"
		element.Range = fmt.Sprintf("%d.%d.x", int(element.Major), int(element.Minor))
		versionDetailsArr = append(versionDetailsArr, element)
	}
	stablesStringArr := storkutil.SortSemversAsc(&stablesArr)
	return versionDetailsArr, stablesStringArr
}
