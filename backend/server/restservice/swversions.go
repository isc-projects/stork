package restservice

import (
	"fmt"

	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

type OfflineJsonVersionDetails struct {

	// eol date
	EolDate string `json:"eolDate,omitempty"`

	// esv
	Esv string `json:"esv,omitempty"`

	// release date
	// Required: true
	ReleaseDate *string `json:"releaseDate"`

	// status
	Status string `json:"status,omitempty"`

	// version
	// Required: true
	Version storkutil.SemanticVersion `json:"version"`
}

type OfflineJsonAppVersionMetadata struct {

	// current stable
	CurrentStable []*OfflineJsonVersionDetails `json:"currentStable,omitempty"`

	// latest dev
	// Required: true
	LatestDev *OfflineJsonVersionDetails `json:"latestDev,omitempty"`

	// latest secure
	LatestSecure *OfflineJsonVersionDetails `json:"latestSecure,omitempty"`
}

type OfflineJsonAppsVersions struct {

	// bind9
	// Required: true
	Bind9 *OfflineJsonAppVersionMetadata `json:"bind9"`

	// date
	// Required: true
	Date *string `json:"date"`

	// out
	// Required: true
	Kea *OfflineJsonAppVersionMetadata `json:"out"`

	// stork
	// Required: true
	Stork *OfflineJsonAppVersionMetadata `json:"stork"`
}

func AppVersionMetadataToRestAPI(input OfflineJsonAppVersionMetadata) *models.AppVersionMetadata {
	out := models.AppVersionMetadata{}
	if input.LatestSecure != nil {
		out.LatestSecure = VersionDetailsToRestAPI(*input.LatestSecure)
		out.LatestSecure.Status = "Current Stable"
	}
	if input.LatestDev != nil {
		out.LatestDev = VersionDetailsToRestAPI(*input.LatestDev)
		out.LatestDev.Status = "Development"
	}
	if input.CurrentStable != nil {
		out.CurrentStable, out.SortedStables = StableSwVersionsToRestAPI(input.CurrentStable)
	}

	return &out
}

func VersionDetailsToRestAPI(input OfflineJsonVersionDetails) *models.VersionDetails {
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

func StableSwVersionsToRestAPI(input []*OfflineJsonVersionDetails) ([]*models.VersionDetails, []string) {
	versionDetailsArr := []*models.VersionDetails{}
	stablesArr := []string{}

	for _, details := range input {

		element := VersionDetailsToRestAPI(*details)
		stablesArr = append(stablesArr, *element.Version)
		element.Status = "Current Stable"
		element.Range = fmt.Sprintf("%d.%d.x", int(element.Major), int(element.Minor))
		versionDetailsArr = append(versionDetailsArr, element)
	}
	stablesArr, err := storkutil.SortSemverStringsAsc(stablesArr)

	if err != nil {
		return nil, nil
	}

	return versionDetailsArr, stablesArr
}
