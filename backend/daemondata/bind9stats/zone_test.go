package bind9stats

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

//go:embed testdata/zones.json
var bind9Zones []byte

// Test instantiating views.
func TestNewZones(t *testing.T) {
	// Create unsorted list of zones.
	zoneList := []*Zone{
		{
			ZoneName: "my.example.org",
		},
		{
			ZoneName: "yours.example.com",
		},
	}
	zones := newZones(zoneList)
	require.NotNil(t, zones)
	// The zones should be sorted.
	require.Equal(t, []string{"yours.example.com", "my.example.org"}, zones.GetZoneNames())
}

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

	index := linearSearchZone(zones.zoneList, "authors.bind")
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name())
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test searching a zone lower bound using linear search.
func TestLinearSearchLowerBoundEqual(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	index, equal := linearSearchZoneLowerBound(zones.zoneList, "authors.bind")
	require.True(t, equal)
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name())
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test searching a zone lower bound using linear search.
func TestLinearSearchLowerBoundUnequal(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	index, equal := linearSearchZoneLowerBound(zones.zoneList, "abc.authors.bind")
	require.False(t, equal)
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "hostname.bind", zone.Name())
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

	index := binarySearchZone(zones.zoneList, "authors.bind")
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name())
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test searching a zone lower bound using binary search.
func TestBinarySearchLowerBoundEqual(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	index, equal := binarySearchZoneLowerBound(zones.zoneList, "authors.bind")
	require.True(t, equal)
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "authors.bind", zone.Name())
	require.Equal(t, "CH", zone.Class)
	require.Zero(t, zone.Serial)
	require.Equal(t, "builtin", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test searching a zone lower bound using binary search.
func TestBinarySearchLowerBoundUnequal(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	index, equal := binarySearchZoneLowerBound(zones.zoneList, "abc.authors.bind")
	require.False(t, equal)
	require.GreaterOrEqual(t, index, 0)
	zone := zones.zoneList[index]
	require.NotNil(t, zone)

	require.Equal(t, "hostname.bind", zone.Name())
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

// Test getting the number of zones.
func TestGetZoneCount(t *testing.T) {
	var zones zones
	err := json.Unmarshal(bind9Zones, &zones)
	require.NoError(t, err)

	count := zones.GetZoneCount()
	require.EqualValues(t, 4, count)
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

	require.Equal(t, "authors.bind", zone.Name())
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
func generateRandomZones(num int) zones {
	generatedZones := testutil.GenerateRandomZones(num)
	var zones zones
	for _, generatedZone := range generatedZones {
		zones.zoneList = append(zones.zoneList, &Zone{
			ZoneName: generatedZone.Name,
			Class:    generatedZone.Class,
			Serial:   generatedZone.Serial,
			Type:     generatedZone.Type,
		})
	}
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
	testCases := []int{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.zoneList[rand.IntN(len(zones.zoneList))]
				binarySearchZone(zones.zoneList, zone.Name())
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
	testCases := []int{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.zoneList[rand.IntN(len(zones.zoneList))]
				linearSearchZone(zones.zoneList, zone.Name())
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
	testCases := []int{10, 100, 1000, 3000, 10000, 100000}
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			zones := generateRandomZones(testCase)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				zone := zones.zoneList[rand.IntN(len(zones.zoneList))]
				zones.GetZone(zone.Name())
			}
		})
	}
}
