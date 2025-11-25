package bind9stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that all zones are returned when filter is nil.
func TestApplyZoneLowerBoundFilterNil(t *testing.T) {
	zoneNames := []string{"example.com", "subdomain.example.com", "example.org", "subdomain.example.org"}
	var zones []*Zone
	for _, name := range zoneNames {
		zones = append(zones, &Zone{
			ZoneName: name,
		})
	}
	filteredZones := ApplyZoneLowerBoundFilter(zones, nil)
	require.Len(t, filteredZones, 4)
	require.ElementsMatch(t, filteredZones, zones)
}

// Test that subset of zones is returned according to the lower bound
// filtering rules.
func TestApplyZoneLowerBoundFilter(t *testing.T) {
	zoneNames := []string{"example.com", "subdomain.example.com", "example.org", "subdomain.example.org"}
	var zones []*Zone
	for _, name := range zoneNames {
		zones = append(zones, &Zone{
			ZoneName: name,
		})
	}
	filter := NewZoneFilter()

	// First page.
	filter.SetLowerBound("", 2)
	filteredZones := ApplyZoneLowerBoundFilter(zones, filter)
	require.Len(t, filteredZones, 4)
	require.ElementsMatch(t, filteredZones, zones)

	// Second page.
	filter.SetLowerBound("subdomain.example.com", 2)
	filteredZones = ApplyZoneLowerBoundFilter(zones, filter)
	require.Len(t, filteredZones, 2)
	require.Equal(t, "example.org", filteredZones[0].Name())
	require.Equal(t, "subdomain.example.org", filteredZones[1].Name())

	// Out of bounds page.
	filter.SetLowerBound("subdomain.example.org", 2)
	filteredZones = ApplyZoneLowerBoundFilter(zones, filter)
	require.Empty(t, filteredZones)

	// Empty lower bound.
	filter.SetLowerBound("", 6)
	filteredZones = ApplyZoneLowerBoundFilter(zones, filter)
	require.Len(t, filteredZones, 4)
	require.ElementsMatch(t, filteredZones, zones)

	// Non existing lower bound between example.com and example.org.
	filter.SetLowerBound("example.gov", 6)
	filteredZones = ApplyZoneLowerBoundFilter(zones, filter)
	require.Len(t, filteredZones, 2)
	require.Equal(t, "example.org", filteredZones[0].Name())
	require.Equal(t, "subdomain.example.org", filteredZones[1].Name())
}

// Test setting filtering by view.
func TestZoneFilterSetView(t *testing.T) {
	filter := NewZoneFilter()
	require.Nil(t, filter.View)

	filter.SetView("_bind")
	require.NotNil(t, filter.View)
	require.Equal(t, "_bind", *filter.View)
}

// Test setting filtering by lower bound zone loading time.
func TestZoneFilterSetLoadedAfter(t *testing.T) {
	filter := NewZoneFilter()
	require.Nil(t, filter.LoadedAfter)

	after := time.Now()
	filter.SetLoadedAfter(after)
	require.NotNil(t, filter.LoadedAfter)
	require.Equal(t, after, *filter.LoadedAfter)
}

// Test setting filtering by page.
func TestZoneFilterSetLowerBound(t *testing.T) {
	filter := NewZoneFilter()
	require.Nil(t, filter.LowerBound)
	require.Nil(t, filter.Limit)

	filter.SetLowerBound("example.org", 3)
	require.NotNil(t, filter.LowerBound)
	require.NotNil(t, filter.Limit)
	require.EqualValues(t, "example.org", *filter.LowerBound)
	require.EqualValues(t, 3, *filter.Limit)
}

// Test setting the limit and offset.
func TestZoneFilterLimit(t *testing.T) {
	filter := NewZoneFilter()
	require.Nil(t, filter.LowerBound)
	require.Nil(t, filter.Limit)

	filter.SetOffsetLimit(1, 3)
	require.EqualValues(t, 1, *filter.Offset)
	require.EqualValues(t, 3, *filter.Limit)
	filter.SetOffsetLimit(2, 5)
	require.EqualValues(t, 2, *filter.Offset)
	require.EqualValues(t, 5, *filter.Limit)
}
