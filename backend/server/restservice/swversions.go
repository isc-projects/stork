package restservice

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// Structs used to deserialize offline versions.json report.

// This struct represents details about either stable, development or security software release.
// ReleaseDate and Version are mandatory.
type ReportVersionDetails struct {
	EolDate     string                    `json:"eolDate,omitempty"`
	Esv         string                    `json:"esv,omitempty"`
	ReleaseDate *string                   `json:"releaseDate"`
	Version     storkutil.SemanticVersion `json:"version"`
}

// This struct gathers different types of releases and contains details for each type.
type ReportAppVersionMetadata struct {
	CurrentStable []*ReportVersionDetails `json:"currentStable,omitempty"`
	LatestDev     *ReportVersionDetails   `json:"latestDev,omitempty"`
	LatestSecure  []*ReportVersionDetails `json:"latestSecure,omitempty"`
}

// This is top level struct representing all metadata for ISC Kea, BIND9 and Stork latest software releases.
type ReportAppsVersions struct {
	Bind9 *ReportAppVersionMetadata `json:"bind9"`
	Date  *string                   `json:"date"`
	Kea   *ReportAppVersionMetadata `json:"kea"`
	Stork *ReportAppVersionMetadata `json:"stork"`
}

// VersionsJSONPath is a path to a JSON file with offline software versions metadata.
// It needs to be modified by tests so it has to be global variable.
var VersionsJSONPath = "/etc/stork/versions.json" //nolint:gochecknoglobals

// Post processes either Kea, Bind9 or Stork version metadata and returns the data in REST API format.
// It returns an error when problem occurs when parsing dates.
func appVersionMetadataToRestAPI(input ReportAppVersionMetadata) (*models.AppVersionMetadata, error) {
	out := models.AppVersionMetadata{}
	if input.LatestSecure != nil {
		if v, err := secureSoftwareVersionsToRestAPI(input.LatestSecure); err == nil {
			out.LatestSecure = v
		} else {
			return nil, err
		}
	}
	if input.LatestDev != nil && input.LatestDev.ReleaseDate != nil {
		if v, err := versionDetailsToRestAPI(*input.LatestDev); err == nil {
			out.LatestDev = v
			out.LatestDev.Status = "Development"
		} else {
			return nil, err
		}
	}
	if input.CurrentStable != nil {
		if v, stables, err := stableSwVersionsToRestAPI(input.CurrentStable); err == nil {
			out.CurrentStable, out.SortedStableVersions = v, stables
		} else {
			return nil, err
		}
	}
	return &out, nil
}

// Post processes either Kea, Bind9 or Stork software release details and returns the data in REST API format.
// It returns an error when problem occurs when parsing dates.
func versionDetailsToRestAPI(input ReportVersionDetails) (*models.VersionDetails, error) {
	v := input.Version.String()
	relDate := strfmt.Date{}
	if parsedTime, err := time.Parse("2006-01-02", *input.ReleaseDate); err == nil {
		relDate = strfmt.Date(parsedTime)
	} else {
		return nil, errors.Wrapf(err, "failed to parse release date from string %s", *input.ReleaseDate)
	}
	out := models.VersionDetails{
		Version:     &v,
		ReleaseDate: &relDate,
		Major:       int64(input.Version.Major),
		Minor:       int64(input.Version.Minor),
	}
	if len(input.EolDate) > 0 {
		if parsedTime, err := time.Parse("2006-01-02", input.EolDate); err == nil {
			eol := strfmt.Date(parsedTime)
			out.EolDate = &eol
		} else {
			return nil, errors.Wrapf(err, "failed to parse EoL date from string %s", input.EolDate)
		}
	}
	if len(input.Esv) > 0 {
		out.Esv = input.Esv
	}
	return &out, nil
}

// Post processes either Kea, Bind9 or Stork stable release details and returns the data in REST API format.
// Takes an array of pointers to ReportVersionDetails for stable releases.
// Returns an array of pointers to VersionDetails for stable releases in REST API format
// and an array of strings with stable release semvers sorted in ascending order.
// It returns an error when problem occurs when parsing dates.
func stableSwVersionsToRestAPI(input []*ReportVersionDetails) ([]*models.VersionDetails, []string, error) {
	versionDetailsArr := []*models.VersionDetails{}
	stablesArr := []storkutil.SemanticVersion{}

	for _, details := range input {
		if v, err := versionDetailsToRestAPI(*details); err == nil {
			stablesArr = append(stablesArr, details.Version)
			v.Status = "Current Stable"
			v.Range = fmt.Sprintf("%d.%d.x", int(v.Major), int(v.Minor))
			versionDetailsArr = append(versionDetailsArr, v)
		} else {
			return nil, nil, err
		}
	}
	stablesStringArr := storkutil.SortSemversAsc(&stablesArr)
	return versionDetailsArr, stablesStringArr, nil
}

// Post processes either Kea, Bind9 or Stork security release details and returns the data in REST API format.
// Takes an array of pointers to ReportVersionDetails for security releases.
// Returns an array of pointers to VersionDetails for security releases in REST API format.
// It returns an error when problem occurs when parsing dates.
func secureSoftwareVersionsToRestAPI(input []*ReportVersionDetails) ([]*models.VersionDetails, error) {
	versionDetailsArr := []*models.VersionDetails{}

	for _, details := range input {
		if v, err := versionDetailsToRestAPI(*details); err == nil {
			v.Status = "Security update"
			v.Range = fmt.Sprintf("%d.%d.x", int(v.Major), int(v.Minor))
			versionDetailsArr = append(versionDetailsArr, v)
		} else {
			return nil, err
		}
	}
	return versionDetailsArr, nil
}
