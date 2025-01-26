package bind9stats

import (
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/views.json
var bind9Views []byte

// Test instantiating a new view.
func TestNewView(t *testing.T) {
	// Create zones in non-DNS order. We want to make sure they will be sorted.
	zones := []*Zone{
		{
			ZoneName: "my.example.org",
		},
		{
			ZoneName: "yours.example.com",
		},
	}
	view := NewView("foo", zones)
	require.NotNil(t, view)
	require.Equal(t, []string{"yours.example.com", "my.example.org"}, view.GetZoneNames())
}

// Test getting the number of zones in a view.
func TestViewGetZoneCount(t *testing.T) {
	// Create zones in non-DNS order. We want to make sure they will be sorted.
	zones := []*Zone{
		{
			ZoneName: "my.example.org",
		},
		{
			ZoneName: "yours.example.com",
		},
	}
	view := NewView("foo", zones)
	require.NotNil(t, view)
	zoneCount, _ := view.GetZoneCount()
	require.EqualValues(t, 2, zoneCount)
}

// Test instantiating views.
func TestNewViews(t *testing.T) {
	// Create unsorted list of views.
	viewList := []*View{
		{
			Name: "_default",
		},
		{
			Name: "_bind",
		},
	}
	views := NewViews(viewList)
	require.NotNil(t, views)
	// The views should be sorted.
	require.Equal(t, []string{"_bind", "_default"}, views.GetViewNames())
}

// Test that zones can be assigned to the view and they are sorted.
func TestSetZones(t *testing.T) {
	// Create the view without zones.
	view := NewView("foo", nil)
	require.NotNil(t, view)

	// Create zones in non-DNS order. We want to make sure they will be sorted.
	zones := []*Zone{
		{
			ZoneName: "my.example.org",
		},
		{
			ZoneName: "yours.example.com",
		},
	}
	view.SetZones(zones)
	require.Equal(t, []string{"yours.example.com", "my.example.org"}, view.GetZoneNames())
}

// Test marshalling the view into binary form.
func TestMarshalView(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	binary, err := json.Marshal(views)
	require.NoError(t, err)
	require.JSONEq(t, string(bind9Views), string(binary))
}

// Test that correct view names are returned.
func TestGetViewNames(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	names := views.GetViewNames()
	require.Len(t, names, 2)
	require.Contains(t, names, "_default")
	require.Contains(t, names, "_bind")
}

// Test getting the total number of zones in all views.
func TestViewsGetZoneCount(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	count := views.GetZoneCount()
	require.EqualValues(t, 6, count)
}

// Test getting a view by name.
func TestGetView(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	view := views.GetView("_default")
	require.NotNil(t, view)
	require.Equal(t, "_default", view.GetViewName())
	require.Len(t, view.GetZoneNames(), 2)

	zones := view.GetZones()
	require.Len(t, zones, 2)
	require.Equal(t, view.Zones.zoneList, zones)
}

// Test getting an iterator returning the zones belonging to the
// view for different filtering settings.
func TestGetZoneIterator(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	view := views.GetView("_bind")
	require.NotNil(t, view)
	require.Equal(t, "_bind", view.GetViewName())
	require.Len(t, view.GetZoneNames(), 4)

	t.Run("no filter", func(t *testing.T) {
		var zones []*Zone
		iterator := view.GetZoneIterator(nil)
		for zone, err := range iterator {
			require.NoError(t, err)
			zones = append(zones, zone)
		}
		require.Len(t, zones, 4)
	})

	t.Run("limit", func(t *testing.T) {
		var zones []*Zone
		filter := NewZoneFilter()
		filter.SetLowerBound("", 2)
		iterator := view.GetZoneIterator(filter)
		for zone, err := range iterator {
			require.NoError(t, err)
			zones = append(zones, zone)
		}
		require.Len(t, zones, 2)
	})

	t.Run("out of bounds name", func(t *testing.T) {
		var zones []*Zone
		filter := NewZoneFilter()
		filter.SetLowerBound("org.server", 2)
		iterator := view.GetZoneIterator(filter)
		for zone, err := range iterator {
			require.NoError(t, err)
			zones = append(zones, zone)
		}
		require.Empty(t, zones)
	})

	t.Run("loaded after", func(t *testing.T) {
		var zones []*Zone
		filter := NewZoneFilter()
		filter.SetLowerBound("", 2)
		filter.SetLoadedAfter(time.Date(2022, 2, 1, 15, 14, 3, 2, time.UTC))
		iterator := view.GetZoneIterator(filter)
		for zone, err := range iterator {
			require.NoError(t, err)
			zones = append(zones, zone)
		}
		require.Len(t, zones, 2)
	})

	t.Run("loaded before", func(t *testing.T) {
		var zones []*Zone
		filter := NewZoneFilter()
		filter.SetLowerBound("", 2)
		filter.SetLoadedAfter(time.Date(2025, 2, 1, 15, 14, 3, 2, time.UTC))
		iterator := view.GetZoneIterator(filter)
		for zone, err := range iterator {
			require.NoError(t, err)
			zones = append(zones, zone)
		}
		require.Empty(t, zones)
	})
}

// Test that nil is returned when the view doesn't exist.
func TestGetViewNonExisting(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	require.Nil(t, views.GetView("non-existent"))
}

// Test that nil is returned when trying to get a view by name in the
// empty list of views.
func TestGetViewEmpty(t *testing.T) {
	var views Views
	require.Nil(t, views.GetView("_default"))
}

// Test that a zone gen be accessed by name in the view.
func TestGetZoneFromView(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	view := views.GetView("_default")
	require.NotNil(t, view)

	zone := view.GetZone("example.com")
	require.NotNil(t, zone)
	require.Equal(t, "example.com", zone.Name())
	require.Equal(t, "IN", zone.Class)
	require.EqualValues(t, 2003080800, zone.Serial)
	require.Equal(t, "primary", zone.Type)
	require.NotZero(t, zone.Loaded)
}

// Test that nil is returned when accessing a non-existing zone
// in a view.
func TestGetZoneFromViewNonExisting(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	view := views.GetView("_default")
	require.NotNil(t, view)

	require.Nil(t, view.GetZone("non.existing.zone"))
}

// Test that nil is returned when accessing a zone in an empty view.
func TestGetZoneFromViewEmpty(t *testing.T) {
	view := View{}
	require.Nil(t, view.GetZone("example.com"))
}
