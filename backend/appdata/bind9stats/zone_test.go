package bind9stats

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/zones.json
var bind9Zones []byte

// Test marshalling the zone into binary form.
func TestMarshalZone(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	binary, err := json.Marshal(zones)
	require.NoError(t, err)
	require.JSONEq(t, string(bind9Zones), string(binary))
}

// Test searching a zone using linear search.
func TestLinearSearch(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	zone := zones.linearSearch("authors.bind")
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name)
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test searching a zone using binary search.
func TestBinarySearch(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	zone := zones.binarySearch("authors.bind")
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name)
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test listing zone names.
func TestGetZoneNames(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	names := zones.GetZoneNames()
	require.Len(t, names, 4)
	require.Contains(t, names, "authors.bind")
	require.Contains(t, names, "hostname.bind")
	require.Contains(t, names, "version.bind")
	require.Contains(t, names, "id.server")
}

// Test that empty list of zone names is returned when the zone
// collection is empty.
func TestGetZoneNamesEmpty(t *testing.T) {
	var zones zones
	require.Empty(t, zones.GetZoneNames())
}

// Test getting a zone from a collection of zones.
func TestGetZone(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	zone := zones.GetZone("authors.bind")
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name)
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test that nil is returned when the accessed zone doesn't exist.
func TestGetZoneNonExisting(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	require.Nil(t, zones.GetZone("non.existing.zone"))
}

// Test that nil is returned when trying to access zone by name
// when the zone list is empty.
func TestGetZoneEmpty(t *testing.T) {
	var zones zones
	require.Nil(t, zones.GetZone("example.com"))
}

// This function generates a collection of zones used in the benchmarks.
// The function argument specifies the number of zones to be generated.
func generateRandomZones(num int64) zones {
	// We construct labels from this set of characters.
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var (
		// We want to simulate a use case where zones have child zones, e.g.:
		// example.com, zone1.example.com, etc. This slice will be used to
		// remember some of the previously generated parent zones and we
		// will derive child zones from it.
		labels []string
		zones  zones
	)
	// Generate num number of zones.
	for i := int64(0); i < num; i++ {
		// For every 10th zone we forget previously generated zones
		// and generate completely new one.
		if i%10 == 0 {
			labels = []string{}
		}
		// Generated zone name will be stored here.
		var name string
		// Generate the name that has up to 6 labels.
		labelsCount := rand.IntN(5) + 1
		// Generate the labels.
		for j := 0; j < labelsCount; j++ {
			// Next label.
			var label string
			// If we cached some labels let's use them to generate
			// a child zone. Don't use the cached label if this is
			// the front label. Reusing front label would cause
			// zone name duplicates.
			if j < labelsCount-1 && len(labels) > j {
				label = labels[j]
			} else {
				// Label is not cached for this position, so we have to generate one.
				// The label length is between 1 and 16.
				for k := 0; k < rand.IntN(15)+1; k++ {
					// Get a random set of characters.
					label += string(charset[rand.IntN(len(charset)-1)])
				}
				// Cache this label for generating child zones.
				labels = append(labels, label)
			}
			// If this is not the last label we should add a dot as a separator.
			if j > 0 {
				label += "."
			}
			// Prepend the label and the dot to the name.
			name = label + name
		}
		// Create the zone entry.
		zones.Zones = append(zones.Zones, &Zone{
			Name: name,
		})
	}
	// The zones must be sorted in DNS order.
	zones.sort()
	return zones
}

// This benchmark measures the zone lookup time using binary search for different
// number of zones in the collection. The zones are sorted using the DNS order.
// During the benchmark development we got the following results:
//
// BenchmarkGetZoneBinarySearch/zones-10-12          4366 ns/op
// BenchmarkGetZoneBinarySearch/zones-100-12         6309 ns/op
// BenchmarkGetZoneBinarySearch/zones-1000-12        10280 ns/op
// BenchmarkGetZoneBinarySearch/zones-3000-12        11824 ns/op
// BenchmarkGetZoneBinarySearch/zones-10000-12       13868 ns/op
// BenchmarkGetZoneBinarySearch/zones-100000-12      17417 ns/op
//
// Compared to the BenchmarkGetZoneLinearSearch it clearly shows that the binary
// search is less efficient for smaller zone collections. It becomes faster
// for larger collections. It is 50x faster for 100000 zones.
func BenchmarkGetZoneBinarySearch(b *testing.B) {
	testCases := []int64{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.Zones[rand.IntN(len(zones.Zones))]
				zones.binarySearch(zone.Name)
			}
		})
	}
}

// This benchmark measures the zone lookup time using binary search for different
// number of zones in the collection. The zones are sorted using the DNS order.
// During the benchmark development we got the following results:
//
// BenchmarkGetZoneLinearSearch/zones-10-12            116.4 ns/op
// BenchmarkGetZoneLinearSearch/zones-100-12           442.6 ns/op
// BenchmarkGetZoneLinearSearch/zones-1000-12         3828 ns/op
// BenchmarkGetZoneLinearSearch/zones-3000-12        12359 ns/op
// BenchmarkGetZoneLinearSearch/zones-10000-12       49160 ns/op
// BenchmarkGetZoneLinearSearch/zones-100000-12     855619 ns/op
//
// Compared to the BenchmarkGetZoneBinarySearch it clearly shows that the linear
// search is way more efficient for smaller zone collections. It is 40x faster for
// 10 zones in the collection. This advantage disappears for several thousands zones
// where the binary search becomes faster.
func BenchmarkGetZoneLinearSearch(b *testing.B) {
	testCases := []int64{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.Zones[rand.IntN(len(zones.Zones))]
				zones.linearSearch(zone.Name)
			}
		})
	}
}

// This benchmark measures the zone lookup time using adaptive technique.
// It runs a linear zone search for smaller zone collections and a binary
// search for larger collections. During the benchmark development we got
// the following results for different zone collections:
//
// BenchmarkGetZone/zones-10-12           134.7 ns/op
// BenchmarkGetZone/zones-100-12          462.8 ns/op
// BenchmarkGetZone/zones-1000-12        3843 ns/op
// BenchmarkGetZone/zones-3000-12       11759 ns/op
// BenchmarkGetZone/zones-10000-12      13718 ns/op
// BenchmarkGetZone/zones-100000-12     17747 ns/op
//
// The benchmark results are the evidence that the GetZone() function takes
// advantage from both searching techniques.
func BenchmarkGetZone(b *testing.B) {
	testCases := []int64{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.Zones[rand.IntN(len(zones.Zones))]
				zones.GetZone(zone.Name)
			}
		})
	}
}
