package pdnsdata

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/zones.json
var pdnsZones []byte

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
	zones := NewZones(zoneList)
	require.NotNil(t, zones)
	// The zones should be sorted.
	require.Len(t, zones.zoneList, 2)
	require.Equal(t, "yours.example.com", zones.zoneList[0].Name())
	require.Equal(t, "my.example.org", zones.zoneList[1].Name())
}

// Test marshalling the zone into binary form.
func TestMarshalZone(t *testing.T) {
	var zones Zones
	err := json.Unmarshal(pdnsZones, &zones)
	require.NoError(t, err)

	binary, err := json.Marshal(zones)
	require.NoError(t, err)
	require.JSONEq(t, string(pdnsZones), string(binary))
}

// Test unmarshalling the zone from binary form.
func TestUnmarshalZone(t *testing.T) {
	var zones Zones
	err := json.Unmarshal(pdnsZones, &zones)
	require.NoError(t, err)
	require.Equal(t, "pdns.example.com.", zones.zoneList[0].Name())
	require.Equal(t, "master", zones.zoneList[0].Kind)
	require.EqualValues(t, 2024031501, zones.zoneList[0].Serial)
	require.Equal(t, "/api/v1/servers/localhost/zones/pdns.example.com.", zones.zoneList[0].URL)
	require.Equal(t, "pdns.example.org.", zones.zoneList[1].Name())
	require.Equal(t, "master", zones.zoneList[1].Kind)
	require.EqualValues(t, 2024031501, zones.zoneList[1].Serial)
	require.Equal(t, "/api/v1/servers/localhost/zones/pdns.example.org.", zones.zoneList[1].URL)
}

// Test iterating over parsed zones.
func TestGetIterator(t *testing.T) {
	var zones Zones
	err := json.Unmarshal(pdnsZones, &zones)
	require.NoError(t, err)

	var iteratedZones []string
	for zone := range zones.GetIterator() {
		iteratedZones = append(iteratedZones, zone.Name())
	}
	require.Equal(t, []string{"pdns.example.com.", "pdns.example.org."}, iteratedZones)
}
