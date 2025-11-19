package dbmodel

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test convenience function enabling filtering by zone types on the GetZonesFilter.
func TestGetZonesFilterEnableZoneTypes(t *testing.T) {
	filter := &GetZonesFilter{}
	filter.EnableZoneType(ZoneTypeSlave)
	filter.EnableZoneType(ZoneTypeBuiltin)
	require.ElementsMatch(t, []ZoneType{ZoneTypeSecondary, ZoneTypeSlave, ZoneTypeBuiltin}, slices.Collect(filter.Types.GetEnabled()))
}

// Test that the zone type filter can be enabled and enabled types can be retrieved.
func TestGetZonesFilterZoneTypes(t *testing.T) {
	filter := NewGetZonesFilterZoneTypes()
	filter.Enable(ZoneTypeSecondary)
	filter.Enable(ZoneTypeDelegationOnly)
	require.ElementsMatch(t, []ZoneType{ZoneTypeSecondary, ZoneTypeDelegationOnly}, slices.Collect(filter.GetEnabled()))
}

// Test that no zone type filters are returned when none are enabled.
func TestGetZonesFilterZoneTypesEmpty(t *testing.T) {
	filter := NewGetZonesFilterZoneTypes()
	require.Nil(t, slices.Collect(filter.GetEnabled()))
}

// Test that the zone type filter can be checked if any filter is specified.
func TestGetZonesFilterZoneTypesIsAnySpecified(t *testing.T) {
	filter := NewGetZonesFilterZoneTypes()
	require.False(t, filter.IsAnySpecified())
	filter.Enable(ZoneTypeSecondary)
	require.True(t, filter.IsAnySpecified())
	filter.Enable(ZoneTypeDelegationOnly)
	require.True(t, filter.IsAnySpecified())
}

// Test inserting and overriding the zones in the database.
func TestAddZonesOverlap(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add two daemons that share zone information.
	var daemons []*Daemon
	for i := 0; i < 2; i++ {
		daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    int64(8000 + i),
			},
		})
		err = AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
	}

	randomZones := testutil.GenerateRandomZones(100)

	// Add zones to the database and associate them with first server.
	var zones []*Zone
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[0].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = AddZones(db, zones...)
	require.NoError(t, err)

	// Make sure that the zones have been added and are associated with one daemon.
	zones, total, err := GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Equal(t, 100, total)
	require.Len(t, zones, 100)
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 1)
		require.Equal(t, zone.LocalZones[0].DaemonID, daemons[0].ID)
	}

	// This time associate the same zones with another server.
	zones = []*Zone{}
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[1].ID,
					View:     "_bind",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = AddZones(db, zones...)
	require.NoError(t, err)

	// Retrieve the zones and their associations. They should be now associated
	// with two servers.
	zones, total, err = GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Equal(t, 100, total)
	require.Len(t, zones, 100)
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 2)
		require.NotEqual(t, zone.LocalZones[0].DaemonID, zone.LocalZones[1].DaemonID)
		require.NotEqual(t, zone.LocalZones[0].View, zone.LocalZones[1].View)
	}
}

// Test getting zones with and without filtering.
func TestGetZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Store zones in the database and associate them with our daemon.
	randomZones := testutil.GenerateRandomZones(25)
	randomZones = testutil.GenerateMoreZonesWithClass(randomZones, 25, "CH")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "secondary")
	randomZones = testutil.GenerateMoreZonesWithSerial(randomZones, 25, 123456)
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "master")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "slave")

	var zones []*Zone
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemon.ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = AddZones(db, zones...)
	require.NoError(t, err)

	t.Run("no filtering", func(t *testing.T) {
		// Without filtering we should get all zones.
		zones, total, err := GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
	})

	t.Run("relations", func(t *testing.T) {
		// Include a daemon table.
		zones, total, err := GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZonesMachine)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)

		for _, zone := range zones {
			require.Len(t, zone.LocalZones, 1)
			require.NotNil(t, zone.LocalZones[0].Daemon)
			require.NotNil(t, zone.LocalZones[0].Daemon.Machine)
		}
	})

	t.Run("filter by serial", func(t *testing.T) {
		filter := &GetZonesFilter{
			Serial: storkutil.Ptr("12345"),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 25, total)
		require.Len(t, zones, 25)
		for _, zone := range zones {
			require.EqualValues(t, 123456, zone.LocalZones[0].Serial)
		}
	})

	t.Run("filter by class", func(t *testing.T) {
		filter := &GetZonesFilter{
			Class: storkutil.Ptr("IN"),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 125, total)
		require.Len(t, zones, 125)
		for _, zone := range zones {
			require.Equal(t, "IN", zone.LocalZones[0].Class)
		}
	})

	t.Run("filter by single zone type", func(t *testing.T) {
		filter := &GetZonesFilter{}
		filter.EnableZoneType(ZoneTypeSecondary)
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		// It should return both secondary and slave, as they are aliases.
		require.Equal(t, 50, total)
		require.Len(t, zones, 50)
		for _, zone := range zones {
			require.Contains(t, []string{"secondary", "slave"}, zone.LocalZones[0].Type)
		}
	})

	t.Run("filter by multiple zone types", func(t *testing.T) {
		filter := &GetZonesFilter{}
		filter.EnableZoneType(ZoneTypeBuiltin)
		filter.EnableZoneType(ZoneTypePrimary)
		filter.EnableZoneType(ZoneTypeSecondary)
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		// It should also include master and slave, as they are aliases of
		// primary and secondary.
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)

		// Collect unique zone types from the zones.
		collectedZoneTypes := make(map[ZoneType]struct{})
		for _, zone := range zones {
			collectedZoneTypes[ZoneType(zone.LocalZones[0].Type)] = struct{}{}
		}
		// There should be two zone types. There is no builtin zone.
		require.Equal(t, 4, len(collectedZoneTypes))
		require.Contains(t, collectedZoneTypes, ZoneTypePrimary)
		require.Contains(t, collectedZoneTypes, ZoneTypeSecondary)
		require.Contains(t, collectedZoneTypes, ZoneTypeMaster)
		require.Contains(t, collectedZoneTypes, ZoneTypeSlave)
	})

	t.Run("filter for zone types unspecified", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)

		// Collect unique zone types from the zones.
		collectedZoneTypes := make(map[ZoneType]struct{})
		for _, zone := range zones {
			collectedZoneTypes[ZoneType(zone.LocalZones[0].Type)] = struct{}{}
		}
		// There should be two zone types. There is no builtin zone.
		require.Equal(t, 4, len(collectedZoneTypes))
		require.Contains(t, collectedZoneTypes, ZoneTypePrimary)
		require.Contains(t, collectedZoneTypes, ZoneTypeSecondary)
	})

	t.Run("lower bound", func(t *testing.T) {
		// Get first 30 zones ordered by DNS name.
		filter := &GetZonesFilter{
			Limit: storkutil.Ptr(30),
		}
		// Let's use default sorting used by restservice.GetZones, i.e. by rname in ascending order.
		zones1, total, err := GetZones(db, filter, string(RName), SortDirAsc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones1, 30)

		// Use the 29th zone as a start (lower bound) for another fetch.
		filter.LowerBound = storkutil.Ptr(zones1[28].Name)
		filter.Limit = storkutil.Ptr(20)
		// Let's use default sorting used by restservice.GetZones, i.e. by rname in ascending order.
		zones2, total, err := GetZones(db, filter, string(RName), SortDirAsc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 121, total)
		require.Len(t, zones2, 20)

		// The first returned zone should overlap with the last zone
		// returned during the first fetch.
		require.Equal(t, zones1[29].Name, zones2[0].Name)
	})

	t.Run("offset", func(t *testing.T) {
		// Get first 20 zones ordered by DNS name.
		filter := &GetZonesFilter{
			Offset: storkutil.Ptr(0),
			Limit:  storkutil.Ptr(20),
		}
		zones1, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones1, 20)

		// Use the 20th zone as a start for another fetch.
		filter.Offset = storkutil.Ptr(19)
		filter.Limit = storkutil.Ptr(20)
		zones2, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones2, 20)

		// The first returned zone should overlap with the last zone
		// returned during the first fetch.
		require.Equal(t, zones1[19].Name, zones2[0].Name)
	})

	t.Run("default sort", func(t *testing.T) {
		filter := &GetZonesFilter{}
		// Let's use default sorting used by restservice.GetZones, i.e. by rname in ascending order.
		zones, total, err := GetZones(db, filter, string(RName), SortDirAsc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		for i := range zones {
			if i > 0 {
				// Compare the current zone with the previous zone. The current zone must
				// always be ordered after the previous.
				require.Negative(t, storkutil.CompareNames(zones[i-1].Name, zones[i].Name))
			}
		}
	})
}

// Test getting zones with sorting applied.
func TestGetZonesWithSorting(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeBind9,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	// Store zones in the database and associate them with our app.
	randomZones := testutil.GenerateRandomZones(25)
	randomZones = testutil.GenerateMoreZonesWithClass(randomZones, 25, "CH")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "secondary")
	randomZones = testutil.GenerateMoreZonesWithSerial(randomZones, 25, 123456)
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "master")
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, "slave")
	// Make zone serials random.
	randomZones = testutil.RandomizeZoneSerials(randomZones, 20251119)

	var zones []*Zone
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: addedDaemons[0].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = AddZones(db, zones...)
	require.NoError(t, err)

	t.Run("sort by rname desc", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, string(RName), SortDirDesc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		for i := range zones {
			if i > 0 {
				// Compare the current zone with the previous zone. The current zone must
				// always be ordered before the previous.
				require.Negative(t, storkutil.CompareNames(zones[i].Name, zones[i-1].Name))
			}
		}
	})

	t.Run("sort by serial asc", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, string(LocalZoneSerial), SortDirAsc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		for i := range zones {
			if i > 0 {
				// Compare the current zone with the previous zone. The current zone must
				// always have serial greater or equal than previous.
				require.GreaterOrEqual(t, zones[i].LocalZones[0].Serial, zones[i-1].LocalZones[0].Serial)
			}
		}
	})

	t.Run("sort by serial desc", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, string(LocalZoneSerial), SortDirDesc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		for i := range zones {
			if i > 0 {
				// Compare the current zone with the previous zone. The current zone must
				// always have serial lower or equal than previous.
				require.LessOrEqual(t, zones[i].LocalZones[0].Serial, zones[i-1].LocalZones[0].Serial)
			}
		}
	})

	t.Run("sort by zone type asc", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, string(LocalZoneType), SortDirAsc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		// Zones should be sorted by local zone type.
		require.Equal(t, "master", zones[0].LocalZones[0].Type)
		require.Equal(t, "primary", zones[25].LocalZones[0].Type)
		require.Equal(t, "primary", zones[50].LocalZones[0].Type)
		require.Equal(t, "primary", zones[75].LocalZones[0].Type)
		require.Equal(t, "secondary", zones[100].LocalZones[0].Type)
		require.Equal(t, "slave", zones[125].LocalZones[0].Type)
		require.Equal(t, "slave", zones[149].LocalZones[0].Type)
	})

	t.Run("sort by zone type desc", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, string(LocalZoneType), SortDirDesc, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 150, total)
		require.Len(t, zones, 150)
		// Zones should be sorted by local zone type in descending order.
		require.Equal(t, "slave", zones[0].LocalZones[0].Type)
		require.Equal(t, "secondary", zones[25].LocalZones[0].Type)
		require.Equal(t, "primary", zones[50].LocalZones[0].Type)
		require.Equal(t, "primary", zones[75].LocalZones[0].Type)
		require.Equal(t, "primary", zones[100].LocalZones[0].Type)
		require.Equal(t, "master", zones[125].LocalZones[0].Type)
		require.Equal(t, "master", zones[149].LocalZones[0].Type)
	})
}

// Test getting zones with filtering by root zone.
func TestGetZonesFilterByRootZone(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Store zones in the database and associate them with our daemon.
	var zones []*Zone
	rootZone := &Zone{
		Name:       ".",
		LocalZones: []*LocalZone{},
	}
	anotherZone := &Zone{
		Name:       "example.com",
		LocalZones: []*LocalZone{},
	}
	zones = append(zones, rootZone, anotherZone)
	for _, zone := range zones {
		zone.LocalZones = append(zone.LocalZones, &LocalZone{
			DaemonID: daemon.ID,
			View:     "_default",
			Class:    "IN",
			Serial:   123456,
			Type:     "primary",
			LoadedAt: time.Now().UTC(),
		})
	}

	err = AddZones(db, zones...)
	require.NoError(t, err)

	t.Run("filter by root zone", func(t *testing.T) {
		searchKeys := []string{"r", "ro", "roo", "root", "(root", "(root)", "R", "Root", "(rooT"}
		for _, searchKey := range searchKeys {
			filter := &GetZonesFilter{
				Text: storkutil.Ptr(searchKey),
			}
			zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
			require.NoError(t, err)
			require.Equal(t, 1, total)
			require.Len(t, zones, 1)
			require.Equal(t, ".", zones[0].Name)
		}
	})

	t.Run("filter by dot", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("."),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, zones, 2)
		require.Equal(t, ".", zones[0].Name)
	})

	t.Run("filter by another zone", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("example"),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example.com", zones[0].Name)
	})
}

// Test getting zones with daemon ID filter.
func TestGetZonesWithDaemonIDFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add several daemons.
	var daemons []*Daemon
	for i := 0; i < 3; i++ {
		daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    int64(8000 + i),
			},
		})
		err = AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
	}

	// Generate random zones and associate them with the daemons.
	randomZones := testutil.GenerateRandomZones(75)
	for i, randomZone := range randomZones {
		daemonID := daemons[i%len(daemons)].ID
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemonID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		}
		err = AddZones(db, zone)
		require.NoError(t, err)
	}

	// Sort daemons by daemon ID to ensure that the last one has the highest ID.
	// When we increase this ID by 1 we should get non-existing ID and
	// no zones should be returned.
	sort.Slice(daemons, func(i, j int) bool {
		return daemons[i].ID < daemons[j].ID
	})

	// Make sure that the zones are returned for each daemon.
	for i := 0; i < 3; i++ {
		filter := &GetZonesFilter{
			DaemonID: storkutil.Ptr(daemons[i].ID),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 25, total)
		require.Len(t, zones, 25)
		for _, zone := range zones {
			require.Equal(t, daemons[i].ID, zone.LocalZones[0].DaemonID)
		}
	}

	// Make sure that the zones are not returned for non-existing daemon ID.
	filter := &GetZonesFilter{
		DaemonID: storkutil.Ptr(daemons[2].ID + 1),
	}
	zones, total, err := GetZones(db, filter, "", SortDirAny,)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// Test getting zones with daemon ID filter when all daemons and some views
// shared the same zones.
func TestGetZonesWithDaemonIDFilterOverlappingZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add several daemons.
	var daemons []*Daemon
	for i := 0; i < 3; i++ {
		daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    int64(8000 + i),
			},
		})
		err = AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
	}

	// Generate random zones and associate them with the daemons.
	randomZones := testutil.GenerateRandomZones(75)
	for _, randomZone := range randomZones {
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[0].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: daemons[0].ID,
					View:     "trusted",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: daemons[1].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: daemons[2].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		}
		err = AddZones(db, zone)
		require.NoError(t, err)
	}

	// Make sure that the zones are returned for each app.
	for i := 0; i < 3; i++ {
		filter := &GetZonesFilter{
			DaemonID: storkutil.Ptr(daemons[i].ID),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 75, total)
		require.Len(t, zones, 75)
		for _, zone := range zones {
			require.Len(t, zone.LocalZones, 4)
		}
	}
}

// Test getting zones with flexible filtering using text.
func TestGetZonesWithTextFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "localhost",
				Port:    int64(8000 + i),
			},
		})
		err = AddDaemon(db, daemon)
		require.NoError(t, err)

		zone := &Zone{
			Name: fmt.Sprintf("example%d.org", i),
			LocalZones: []*LocalZone{
				{
					DaemonID: daemon.ID,
					View:     fmt.Sprintf("view%d", i),
					Class:    "IN",
					Serial:   123456,
					Type:     "primary",
					LoadedAt: time.Now().UTC(),
				},
			},
		}
		err = AddZones(db, zone)
		require.NoError(t, err)
	}

	t.Run("filter by zone name", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("mple0.org"),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example0.org", zones[0].Name)
	})

	t.Run("filter by daemon name", func(t *testing.T) {
		filter := &GetZonesFilter{
			DaemonName: storkutil.Ptr(daemonname.Bind9),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, zones, 3)
		require.Equal(t, "example0.org", zones[0].Name)
		require.Equal(t, "example1.org", zones[1].Name)
	})

	t.Run("filter by view", func(t *testing.T) {
		filter := &GetZonesFilter{
			DaemonName: storkutil.Ptr(daemonname.Bind9),
			Text:       storkutil.Ptr("ew2"),
		}
		zones, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example2.org", zones[0].Name)
	})

	t.Run("match all zone names", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("exam"),
		}
		_, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
	})

	t.Run("match all views", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("vi"),
		}
		_, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
	})

	t.Run("combined filtering", func(t *testing.T) {
		filter := &GetZonesFilter{
			DaemonName: storkutil.Ptr(daemonname.Bind9),
			Text:       storkutil.Ptr("mple0.org"),
		}
		_, total, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
	})
}

// Test getting the number of distinct and builtin zones for a given daemon.
func TestGetZoneCountStatsByDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Store zones in the database and associate them with our app.
	randomZones := testutil.GenerateRandomZones(25)
	randomZones = testutil.GenerateMoreZonesWithType(randomZones, 25, string(ZoneTypeBuiltin))

	// Make sure that the builtin zones are last.
	sort.Slice(randomZones, func(i, j int) bool {
		return randomZones[i].Type > randomZones[j].Type
	})

	for i := 0; i < 10; i++ {
		// Add overlapping zones using a sliding window between i and len(randomZones)-10+i (exclusive).
		// It should result in getting 49 distinct zones and 24 builtin zones.
		for _, randomZone := range randomZones[i : len(randomZones)-10+i] {
			zone := &Zone{
				Name: randomZone.Name,
				LocalZones: []*LocalZone{
					{
						DaemonID: daemon.ID,
						View:     fmt.Sprintf("view%d", i),
						Class:    randomZone.Class,
						Serial:   randomZone.Serial,
						Type:     randomZone.Type,
						LoadedAt: time.Now().UTC(),
					},
				},
			}
			err = AddZones(db, zone)
			require.NoError(t, err)
		}
	}

	stats, err := GetZoneCountStatsByDaemon(db, daemon.ID)
	require.NoError(t, err)
	require.Equal(t, int64(49), stats.DistinctZones)
	require.Equal(t, int64(24), stats.BuiltinZones)
}

// Test getting a zone by its ID.
func TestGetZoneByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	zone := &Zone{
		Name: "example.org",
		LocalZones: []*LocalZone{
			{
				DaemonID: daemon.ID,
				View:     "_default",
				Class:    "IN",
				Serial:   123456,
				Type:     "primary",
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = AddZones(db, zone)
	require.NoError(t, err)

	// Get zone by valid ID.
	zone, err = GetZoneByID(db, zone.ID, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.NotNil(t, zone)
	require.Equal(t, "example.org", zone.Name)

	// Get zone by invalid ID.
	zone, err = GetZoneByID(db, zone.ID+1, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Nil(t, zone)
}

// Test deleting the zones that have no associations with the daemons.
func TestDeleteOrphanedZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	randomZones := testutil.GenerateRandomZones(100)

	// Add the zones to the database.
	var zones []*Zone
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemon.ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}
	err = AddZones(db, zones...)
	require.NoError(t, err)

	// No zones are orphaned so this operation should return zero affected rows.
	affectedRows, err := DeleteOrphanedZones(db)
	require.NoError(t, err)
	require.Zero(t, affectedRows)

	// Remove associations of the daemon with our zones.
	err = DeleteLocalZones(db, daemon.ID)
	require.NoError(t, err)

	// This time all zones are orphaned, so they should get removed.
	affectedRows, err = DeleteOrphanedZones(db)
	require.NoError(t, err)
	require.EqualValues(t, 100, affectedRows)

	// No zones present.
	zones, total, err := GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// Test updating the timestamp of the last zone transfer.
func TestUpdateLocalZoneRRsTransferAt(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	zone := &Zone{
		Name: "example.org",
		LocalZones: []*LocalZone{
			{
				DaemonID: daemon.ID,
				View:     "_default",
				Class:    "IN",
				Serial:   123456,
				Type:     "primary",
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = AddZones(db, zone)
	require.NoError(t, err)

	err = UpdateLocalZoneRRsTransferAt(db, zone.LocalZones[0].ID)
	require.NoError(t, err)

	returnedZone, err := GetZoneByID(db, zone.ID, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Len(t, returnedZone.LocalZones, 1)
	require.NotNil(t, returnedZone.LocalZones[0].ZoneTransferAt)
	require.InDelta(t, time.Now().Unix(), returnedZone.LocalZones[0].ZoneTransferAt.Unix(), 5)
}

// Test getting a local zone from a zone by daemon ID and view.
func TestGetLocalZone(t *testing.T) {
	zone := &Zone{
		Name: "example.org",
		LocalZones: []*LocalZone{
			{
				DaemonID: 1,
				View:     "trusted",
			},
			{
				DaemonID: 2,
				View:     "guest",
			},
		},
	}

	t.Run("matching daemon ID and view", func(t *testing.T) {
		localZone := zone.GetLocalZone(1, "trusted")
		require.NotNil(t, localZone)
	})

	t.Run("matching daemon ID but not view", func(t *testing.T) {
		localZone := zone.GetLocalZone(1, "guest")
		require.Nil(t, localZone)
	})

	t.Run("matching view but not daemon ID", func(t *testing.T) {
		localZone := zone.GetLocalZone(2, "trusted")
		require.Nil(t, localZone)
	})

	t.Run("no matching daemon ID and view", func(t *testing.T) {
		localZone := zone.GetLocalZone(3, "guest")
		require.Nil(t, localZone)
	})
}

// Test the "before insert" hook for the zone.
func TestZoneBeforeInsert(t *testing.T) {
	zone := &Zone{
		Name: "zone.example.org",
	}
	zone.BeforeInsert(context.Background())
	require.Equal(t, "org.example.zone", zone.Rname)
}

// Test the "before update" hook for the zone.
func TestZoneBeforeUpdate(t *testing.T) {
	zone := &Zone{
		Name: "big.zone.example.org",
	}
	zone.BeforeUpdate(context.Background())
	require.Equal(t, "org.example.zone.big", zone.Rname)
}

// This benchmark measures the time to insert zones in batches into the
// database. It inserts 10000 zones using different batch sizes. It is
// an evidence that it is better to insert data in batches rather than
// sequentially. The following results have been obtained during the
// development of this benchmark:
//
// BenchmarkAddZones/zones-1-12         2916306666 ns/op    12013740600 B/op   702690 allocs/op
// BenchmarkAddZones/zones-10-12        504430139 ns/op     1207774253 B/op    135839 allocs/op
// BenchmarkAddZones/zones-100-12       269607531 ns/op     120577492 B/op     84488 allocs/op
// BenchmarkAddZones/zones-1000-12      256385229 ns/op     14314112 B/op      79112 allocs/op
// BenchmarkAddZones/zones-10000-12     257393979 ns/op     7575522 B/op       78552 allocs/op
//
// There is no significant difference between batch size of 100 and 10000.
func BenchmarkAddZones(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	if err != nil {
		b.Fatal(err)
	}

	// Add a daemon.
	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	if err != nil {
		b.Fatal(err)
	}

	// Generate random zones and store them in the database.
	// The test cases will override these zones. That way all
	// benchmarks will have the same database initial state.
	// Otherwise we'd have to remove all zones before each
	// benchmark.
	randomZones := testutil.GenerateRandomZones(10000)
	var zones []*Zone
	for _, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemon.ID,
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					View:     "_default",
					LoadedAt: time.Now().UTC(),
				},
			},
		})
	}

	// Test single zone at the time, and a batch of 10, 100, 1000 and 10000 zones.
	testCases := []int{1, 10, 100, 1000, 10000}

	// Run benchmarks for these test cases.
	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("zones-%d", testCase), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Insert all 10000 zones into the database but using
				// different batch sizes.
				for j := 0; j < 10000; j += testCase {
					err = AddZones(db, zones[j:j+testCase]...)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

// The benchmark measures the time to return 100000 zones from the database.
// The benchmark gave the following result:
//
// BenchmarkGetZones-12   1442389750 ns/op
//
// Changing the number of returned zones affected the time proportionally.
// Testing the implementation using raw SQL queries instead of go-pg ORM
// didn't show any performance improvement.
func BenchmarkGetZones(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	zonesNum := 100000
	randomZones := testutil.GenerateRandomZones(zonesNum)

	var daemons []*Daemon
	for i := range randomZones {
		// Each server holds 1000 zones.
		if i%(zonesNum/1000) == 0 {
			machine := &Machine{
				ID:        0,
				Address:   "localhost",
				AgentPort: int64(8080 + i),
			}
			err := AddMachine(db, machine)
			if err != nil {
				b.Fatal(err)
			}

			daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    int64(8000 + i),
				},
			})
			err = AddDaemon(db, daemon)
			if err != nil {
				b.Fatal(err)
			}

			daemons = append(daemons, daemon)
		}
	}
	// Add the zones to the database.
	batch := NewBatch(db, 10000, AddZones)
	for i, randomZone := range randomZones {
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[i/(zonesNum/1000)].ID,
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					View:     "_default",
					LoadedAt: time.Now().UTC(),
				},
			},
		}
		err := batch.Add(zone)
		if err != nil {
			b.Fatal(err)
		}
	}
	err := batch.Flush()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zones, _, err := GetZones(db, nil, "", SortDirAny, ZoneRelationLocalZones)
		if err != nil {
			b.Fatal(err)
		}
		if len(zones) != zonesNum {
			b.Fatalf("invalid number of zones returned %d", len(zones))
		}
	}
}

// The benchmark measures the time to return 100000 zones from the database
// when filtering by zone type is enabled. The benchmark gave the following
// results:
//
// BenchmarkGetZonesWithZoneTypeFilter-12   2051111792 ns/op
//
// This result is slower than the benchmark for the unfiltered zones because
// we have to join the local_zones table for filtering. However, the performance
// is acceptable.
func BenchmarkGetZonesWithZoneTypeFilter(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	zonesNum := 100000
	randomZones := testutil.GenerateRandomZones(zonesNum)

	var daemons []*Daemon
	for i := range randomZones {
		// Each server holds 1000 zones.
		if i%(zonesNum/1000) == 0 {
			machine := &Machine{
				ID:        0,
				Address:   "localhost",
				AgentPort: int64(8080 + i),
			}
			err := AddMachine(db, machine)
			if err != nil {
				b.Fatal(err)
			}

			daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    int64(8000 + i),
				},
			})
			err = AddDaemon(db, daemon)
			if err != nil {
				b.Fatal(err)
			}

			daemons = append(daemons, daemon)
		}
	}
	// Add the zones to the database.
	batch := NewBatch(db, 10000, AddZones)
	for i, randomZone := range randomZones {
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[i/(zonesNum/1000)].ID,
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					View:     "_default",
					LoadedAt: time.Now().UTC(),
				},
			},
		}
		err := batch.Add(zone)
		if err != nil {
			b.Fatal(err)
		}
	}
	err := batch.Flush()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter := &GetZonesFilter{}
		filter.EnableZoneType(ZoneTypeBuiltin)
		filter.EnableZoneType(ZoneTypeSecondary)
		filter.EnableZoneType(ZoneTypeForward)
		filter.EnableZoneType(ZoneTypeHint)
		filter.EnableZoneType(ZoneTypePrimary)
		filter.EnableZoneType(ZoneTypeRedirect)
		filter.EnableZoneType(ZoneTypeStaticStub)
		filter.EnableZoneType(ZoneTypeStub)
		zones, _, err := GetZones(db, filter, "", SortDirAny, ZoneRelationLocalZones)
		if err != nil {
			b.Fatal(err)
		}
		if len(zones) != zonesNum {
			b.Fatalf("invalid number of zones returned %d", len(zones))
		}
	}
}

// The benchmark measures the time to return the number of distinct zones
// for a given daemon. The benchmark gave the following result:
//
// BenchmarkGetDistinctZoneCount-12   285899406 ns/op
//
// It shows a reasonable performance.
func BenchmarkGetDistinctZoneCount(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	zonesNum := 100000
	randomZones := testutil.GenerateRandomZones(zonesNum)

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	if err != nil {
		b.Fatal(err)
	}

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "localhost",
			Port:    8000,
		},
	})
	err = AddDaemon(db, daemon)
	if err != nil {
		b.Fatal(err)
	}

	// Add the zones to the database to different views.
	var views []string
	for i := 0; i < 10; i++ {
		views = append(views, fmt.Sprintf("view%d", i))
	}

	batch := NewBatch(db, 10000, AddZones)
	// The zones in each view overlap.
	for _, view := range views {
		for _, randomZone := range randomZones {
			zone := &Zone{
				Name: randomZone.Name,
				LocalZones: []*LocalZone{
					{
						DaemonID: daemon.ID,
						Class:    randomZone.Class,
						Serial:   randomZone.Serial,
						Type:     randomZone.Type,
						View:     view,
						LoadedAt: time.Now().UTC(),
					},
				},
			}
			err := batch.Add(zone)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	err = batch.Flush()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats, err := GetZoneCountStatsByDaemon(db, daemon.ID)
		if err != nil {
			b.Fatal(err)
		}
		if stats.DistinctZones != int64(zonesNum) {
			b.Fatalf("invalid number of zones returned %d", stats.DistinctZones)
		}
	}
}
