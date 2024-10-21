package storkutil

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/pkg/errors"
)

// Represents a semantic version number.
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// Creates a new semantic version.
func NewSemanticVersion(major, minor, patch int) SemanticVersion {
	return SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// Returns a string representation of the semantic version.
func (v SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Returns true if the semantic version is less than the other semantic version.
func (v SemanticVersion) LessThan(other SemanticVersion) bool {
	if v.Major == other.Major {
		if v.Minor == other.Minor {
			return v.Patch < other.Patch
		}
		return v.Minor < other.Minor
	}
	return v.Major < other.Major
}

// Returns true if the semantic version is greater than the other semantic version.
func (v SemanticVersion) GreaterThan(other SemanticVersion) bool {
	if v.Major == other.Major {
		if v.Minor == other.Minor {
			return v.Patch > other.Patch
		}
		return v.Minor > other.Minor
	}
	return v.Major > other.Major
}

// Returns true if the semantic version is equal to the other semantic version.
func (v SemanticVersion) Equal(other SemanticVersion) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

// Returns true if the semantic version is less than or equal to the other semantic version.
func (v SemanticVersion) LessThanOrEqual(other SemanticVersion) bool {
	return v.LessThan(other) || v.Equal(other)
}

// Returns true if the semantic version is greater than or equal to the other semantic version.
func (v SemanticVersion) GreaterThanOrEqual(other SemanticVersion) bool {
	return v.GreaterThan(other) || v.Equal(other)
}

// Parses a semantic version string into a SemanticVersion struct.
func ParseSemanticVersion(version string) (SemanticVersion, error) {
	var v SemanticVersion
	_, err := fmt.Sscanf(version, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	return v, errors.Wrap(err, "invalid semantic version")
}

// Parses a semantic version string into a SemanticVersion struct. If the
// version string is invalid, returns a SemanticVersion struct with the maximum
// possible values.
func ParseSemanticVersionOrLatest(version string) SemanticVersion {
	semver, err := ParseSemanticVersion(version)
	if err != nil {
		return NewSemanticVersion(math.MaxInt, math.MaxInt, math.MaxInt)
	}
	return semver
}

// Represents sort interface of semantic versions to be ordered ascending.
type BySemverAsc []SemanticVersion

// sort.Interface methods for sorting semantic versions.

// Returns length of an array of SemanticVersions to be sorted.
func (semvers BySemverAsc) Len() int {
	return len(semvers)
}

// Comparing function. Returns true if i-th semver value in the array of SemanticVersions to be sorted
// is lower than j-th semver value in the same array.
func (semvers BySemverAsc) Less(i, j int) bool {
	return semvers[i].LessThan(semvers[j])
}

// Function swapping i-th and j-th elements in the array of SemanticVersions to be sorted.
func (semvers BySemverAsc) Swap(i, j int) {
	semvers[i], semvers[j] = semvers[j], semvers[i]
}

// Takes an array of SemanticVersions and sorts them
// in ascending order and returns back as an array of version strings.
func SortSemversAsc(semvers *[]SemanticVersion) []string {
	var results []string
	sort.Sort(BySemverAsc(*semvers))
	for _, semver := range *semvers {
		results = append(results, semver.String())
	}
	return results
}

// Deserializes semantic version from JSON string.
func (v *SemanticVersion) UnmarshalJSON(data []byte) error {
	var version string
	err := json.Unmarshal(data, &version)
	if err != nil {
		return errors.Wrap(err, "problem deserializing semantic version from JSON")
	}
	*v, err = ParseSemanticVersion(version)
	if err != nil {
		return err
	}
	return nil
}
