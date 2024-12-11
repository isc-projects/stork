package bind9stats

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/views.json
var bind9Views []byte

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

// Test getting a view by name.
func TestGetView(t *testing.T) {
	var views Views
	err := json.Unmarshal(bind9Views, &views)
	require.NoError(t, err)

	view := views.GetView("_default")
	require.NotNil(t, view)
	require.Equal(t, "_default", view.Name)
	require.Len(t, view.GetZoneNames(), 2)
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
	require.Equal(t, "example.com", zone.Name)
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
