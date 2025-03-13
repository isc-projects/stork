package dbmodel

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test convenience function enabling filtering by zone types on the GetZonesFilter.
func TestGetZonesFilterEnableZoneTypes(t *testing.T) {
	filter := &GetZonesFilter{}
	filter.EnableZoneType(ZoneTypeSecondary)
	filter.EnableZoneType(ZoneTypeBuiltin)
	require.Equal(t, []ZoneType{ZoneTypeSecondary, ZoneTypeBuiltin}, slices.Collect(filter.Types.GetEnabled()))
}

// Test that the zone type filter can be enabled and enabled types can be retrieved.
func TestGetZonesFilterZoneTypes(t *testing.T) {
	filter := NewGetZonesFilterZoneTypes()
	filter.Enable(ZoneTypeSecondary)
	filter.Enable(ZoneTypeDelegationOnly)
	require.Equal(t, []ZoneType{ZoneTypeSecondary, ZoneTypeDelegationOnly}, slices.Collect(filter.GetEnabled()))
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

	// Add two apps that share zone information.
	var apps []*App
	for i := 0; i < 2; i++ {
		apps = append(apps, &App{
			ID:        0,
			MachineID: machine.ID,
			Type:      AppTypeKea,
			Daemons: []*Daemon{
				NewBind9Daemon(true),
			},
		})
		addedDaemons, err := AddApp(db, apps[i])
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)
	}

	randomZones := testutil.GenerateRandomZones(100)

	// Add zones to the database and associate them with first server.
	var zones []*Zone
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: apps[0].Daemons[0].ID,
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

	// Make sure that the zones have been added and are associated with one server.
	zones, total, err := GetZones(db, nil, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Equal(t, 100, total)
	require.Len(t, zones, 100)
	for _, zone := range zones {
		require.Len(t, zone.LocalZones, 1)
		require.Equal(t, zone.LocalZones[0].DaemonID, apps[0].Daemons[0].ID)
	}

	// This time associate the same zones with another server.
	zones = []*Zone{}
	for i, randomZone := range randomZones {
		zones = append(zones, &Zone{
			Name: randomZones[i].Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: apps[1].Daemons[0].ID,
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
	zones, total, err = GetZones(db, nil, ZoneRelationLocalZones)
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

	t.Run("no filtering", func(t *testing.T) {
		// Without filtering we should get all zones.
		zones, total, err := GetZones(db, nil, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones, 100)
	})

	t.Run("relations", func(t *testing.T) {
		// Include daemon and app tables.
		zones, total, err := GetZones(db, nil, ZoneRelationLocalZonesApp)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones, 100)

		for _, zone := range zones {
			require.Len(t, zone.LocalZones, 1)
			require.NotNil(t, zone.LocalZones[0].Daemon)
			require.NotNil(t, zone.LocalZones[0].Daemon.App)
		}
	})

	t.Run("filter by serial", func(t *testing.T) {
		filter := &GetZonesFilter{
			Serial: storkutil.Ptr("12345"),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
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
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 75, total)
		require.Len(t, zones, 75)
		for _, zone := range zones {
			require.Equal(t, "IN", zone.LocalZones[0].Class)
		}
	})

	t.Run("filter by single zone type", func(t *testing.T) {
		filter := &GetZonesFilter{}
		filter.EnableZoneType(ZoneTypeSecondary)
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 25, total)
		require.Len(t, zones, 25)
		for _, zone := range zones {
			require.Equal(t, "secondary", zone.LocalZones[0].Type)
		}
	})

	t.Run("filter by multiple zone types", func(t *testing.T) {
		filter := &GetZonesFilter{}
		filter.EnableZoneType(ZoneTypeBuiltin)
		filter.EnableZoneType(ZoneTypePrimary)
		filter.EnableZoneType(ZoneTypeSecondary)
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones, 100)

		// Collect unique zone types from the zones.
		collectedZoneTypes := make(map[ZoneType]struct{})
		for _, zone := range zones {
			collectedZoneTypes[ZoneType(zone.LocalZones[0].Type)] = struct{}{}
		}
		// There should be two zone types. There is no builtin zone.
		require.Equal(t, 2, len(collectedZoneTypes))
		require.Contains(t, collectedZoneTypes, ZoneTypePrimary)
		require.Contains(t, collectedZoneTypes, ZoneTypeSecondary)
	})

	t.Run("filter for zone types unspecified", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones, 100)

		// Collect unique zone types from the zones.
		collectedZoneTypes := make(map[ZoneType]struct{})
		for _, zone := range zones {
			collectedZoneTypes[ZoneType(zone.LocalZones[0].Type)] = struct{}{}
		}
		// There should be two zone types. There is no builtin zone.
		require.Equal(t, 2, len(collectedZoneTypes))
		require.Contains(t, collectedZoneTypes, ZoneTypePrimary)
		require.Contains(t, collectedZoneTypes, ZoneTypeSecondary)
	})

	t.Run("lower bound", func(t *testing.T) {
		// Get first 30 zones ordered by DNS name.
		filter := &GetZonesFilter{
			Limit: storkutil.Ptr(30),
		}
		zones1, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones1, 30)

		// Use the 29th zone as a start (lower bound) for another fetch.
		filter.LowerBound = storkutil.Ptr(zones1[28].Name)
		filter.Limit = storkutil.Ptr(20)
		zones2, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 71, total)
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
		zones1, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones1, 20)

		// Use the 20th zone as a start for another fetch.
		filter.Offset = storkutil.Ptr(19)
		filter.Limit = storkutil.Ptr(20)
		zones2, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones2, 20)

		// The first returned zone should overlap with the last zone
		// returned during the first fetch.
		require.Equal(t, zones1[19].Name, zones2[0].Name)
	})

	t.Run("sort", func(t *testing.T) {
		filter := &GetZonesFilter{}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 100, total)
		require.Len(t, zones, 100)
		for i := range zones {
			if i > 0 {
				// Compare the current zone with the previous zone. The current zone must
				// always be ordered after the previous.
				require.Negative(t, storkutil.CompareNames(zones[i-1].Name, zones[i].Name))
			}
		}
	})
}

// Test getting zones with app ID filter.
func TestGetZonesWithAppIDFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add several apps.
	var apps []*App
	for i := 0; i < 3; i++ {
		app := &App{
			ID:        0,
			MachineID: machine.ID,
			Type:      AppTypeBind9,
			Name:      fmt.Sprintf("app%d", i),
			Daemons: []*Daemon{
				NewBind9Daemon(true),
			},
		}
		addedDaemons, err := AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)
		apps = append(apps, app)
	}

	// Generate random zones and associate them with the apps.
	randomZones := testutil.GenerateRandomZones(75)
	for i, randomZone := range randomZones {
		daemonID := apps[i%len(apps)].Daemons[0].ID
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

	// Sort apps by app ID to ensure that the last one has the highest ID.
	// When we increase this ID by 1 we should get non-existing ID and
	// no zones should be returned.
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].ID < apps[j].ID
	})

	// Make sure that the zones are returned for each app.
	for i := 0; i < 3; i++ {
		filter := &GetZonesFilter{
			AppID: storkutil.Ptr(apps[i].ID),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZonesApp)
		require.NoError(t, err)
		require.Equal(t, 25, total)
		require.Len(t, zones, 25)
		for _, zone := range zones {
			require.Equal(t, apps[i].ID, zone.LocalZones[0].Daemon.AppID)
		}
	}

	// Make sure that the zones are not returned for non-existing app ID.
	filter := &GetZonesFilter{
		AppID: storkutil.Ptr(apps[2].ID + 1),
	}
	zones, total, err := GetZones(db, filter, ZoneRelationLocalZonesApp)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
}

// Test getting zones with app ID filter when all apps and some views
// shared the same zones.
func TestGetZonesWithAppIDFilterOverlappingZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add several apps.
	var apps []*App
	for i := 0; i < 3; i++ {
		app := &App{
			ID:        0,
			MachineID: machine.ID,
			Type:      AppTypeBind9,
			Name:      fmt.Sprintf("app%d", i),
			Daemons: []*Daemon{
				NewBind9Daemon(true),
			},
		}
		addedDaemons, err := AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)
		apps = append(apps, app)
	}

	// Generate random zones and associate them with the apps.
	randomZones := testutil.GenerateRandomZones(75)
	for _, randomZone := range randomZones {
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: apps[0].Daemons[0].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: apps[0].Daemons[0].ID,
					View:     "trusted",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: apps[1].Daemons[0].ID,
					View:     "_default",
					Class:    randomZone.Class,
					Serial:   randomZone.Serial,
					Type:     randomZone.Type,
					LoadedAt: time.Now().UTC(),
				},
				{
					DaemonID: apps[2].Daemons[0].ID,
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
			AppID: storkutil.Ptr(apps[i].ID),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZonesApp)
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
		app := &App{
			ID:        0,
			MachineID: machine.ID,
			Type:      AppTypeBind9,
			Name:      fmt.Sprintf("app%d", i),
			Daemons: []*Daemon{
				NewBind9Daemon(true),
			},
		}
		addedDaemons, err := AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)

		zone := &Zone{
			Name: fmt.Sprintf("example%d.org", i),
			LocalZones: []*LocalZone{
				{
					DaemonID: addedDaemons[0].ID,
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
			AppType: storkutil.Ptr("bind9"),
			Text:    storkutil.Ptr("mple0.org"),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example0.org", zones[0].Name)
	})

	t.Run("filter by app name", func(t *testing.T) {
		filter := &GetZonesFilter{
			AppType: storkutil.Ptr("bind9"),
			Text:    storkutil.Ptr("pp1"),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example1.org", zones[0].Name)
	})

	t.Run("filter by view", func(t *testing.T) {
		filter := &GetZonesFilter{
			AppType: storkutil.Ptr("bind9"),
			Text:    storkutil.Ptr("ew2"),
		}
		zones, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, zones, 1)
		require.Equal(t, "example2.org", zones[0].Name)
	})

	t.Run("match all zone names", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("exam"),
		}
		_, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
	})

	t.Run("match all app names", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("app"),
		}
		_, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
	})

	t.Run("match all views", func(t *testing.T) {
		filter := &GetZonesFilter{
			Text: storkutil.Ptr("vi"),
		}
		_, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Equal(t, 3, total)
	})

	t.Run("combined filtering", func(t *testing.T) {
		filter := &GetZonesFilter{
			AppType: storkutil.Ptr("kea"),
			Text:    storkutil.Ptr("mple0.org"),
		}
		_, total, err := GetZones(db, filter, ZoneRelationLocalZones)
		require.NoError(t, err)
		require.Zero(t, total)
	})
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

	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	randomZones := testutil.GenerateRandomZones(100)

	// Add the zones to the database.
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

	// No zones are orphaned so this operation should return zero affected rows.
	affectedRows, err := DeleteOrphanedZones(db)
	require.NoError(t, err)
	require.Zero(t, affectedRows)

	// Remove associations of the daemon with our zones.
	err = DeleteLocalZones(db, addedDaemons[0].ID)
	require.NoError(t, err)

	// This time all zones are orphaned, so they should get removed.
	affectedRows, err = DeleteOrphanedZones(db)
	require.NoError(t, err)
	require.EqualValues(t, 100, affectedRows)

	// No zones present.
	zones, total, err := GetZones(db, nil, ZoneRelationLocalZones)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, zones)
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

	// Add an app.
	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
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
					DaemonID: addedDaemons[0].ID,
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

			app := &App{
				ID:        0,
				MachineID: machine.ID,
				Type:      AppTypeKea,
				Daemons: []*Daemon{
					NewBind9Daemon(true),
				},
			}
			addedDaemons, err := AddApp(db, app)
			if err != nil {
				b.Fatal(err)
			}

			daemons = append(daemons, addedDaemons...)
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
		zones, _, err := GetZones(db, nil, ZoneRelationLocalZones)
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

			app := &App{
				ID:        0,
				MachineID: machine.ID,
				Type:      AppTypeKea,
				Daemons: []*Daemon{
					NewBind9Daemon(true),
				},
			}
			addedDaemons, err := AddApp(db, app)
			if err != nil {
				b.Fatal(err)
			}

			daemons = append(daemons, addedDaemons...)
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
		zones, _, err := GetZones(db, filter, ZoneRelationLocalZones)
		if err != nil {
			b.Fatal(err)
		}
		if len(zones) != zonesNum {
			b.Fatalf("invalid number of zones returned %d", len(zones))
		}
	}
}
