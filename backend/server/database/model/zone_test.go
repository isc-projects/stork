package dbmodel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/appdata/bind9stats"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

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
	zones, err = GetZones(db, nil)
	require.NoError(t, err)
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
	zones, err = GetZones(db, nil)
	require.NoError(t, err)
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
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	// Store zones in the database and associate them with our app.
	randomZones := testutil.GenerateRandomZones(100)

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
		zones, err = GetZones(db, nil)
		require.NoError(t, err)
		require.Len(t, zones, 100)
	})

	t.Run("filter by existing view", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_default")
		zones, err = GetZones(db, filter)
		require.NoError(t, err)
		require.Len(t, zones, 100)
	})

	t.Run("filter by non-existing view", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		filter.SetView("_bind")
		zones, err = GetZones(db, filter)
		require.NoError(t, err)
		require.Empty(t, zones)
	})

	t.Run("lower bound", func(t *testing.T) {
		// Get first 30 zones ordered by DNS name.
		filter := bind9stats.NewZoneFilter()
		filter.SetLowerBound("", 30)
		zones1, err := GetZones(db, filter)
		require.NoError(t, err)
		require.Len(t, zones1, 30)

		// Use the 29th zone as a start (lower bound) for another fetch.
		filter.SetLowerBound(zones1[28].Name, 20)
		zones2, err := GetZones(db, filter)
		require.NoError(t, err)
		require.Len(t, zones2, 20)

		// The first returned zone should overlap with the last zone
		// returned during the first fetch.
		require.Equal(t, zones1[29].Name, zones2[0].Name)
	})

	t.Run("sort", func(t *testing.T) {
		filter := bind9stats.NewZoneFilter()
		zones, err = GetZones(db, filter)
		require.NoError(t, err)
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
	zones, err = GetZones(db, nil)
	require.NoError(t, err)
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
// BenchmarkAddZones/zones-1-12         3132826500 ns/op
// BenchmarkAddZones/zones-10-12        458866820 ns/op
// BenchmarkAddZones/zones-100-12       720914292 ns/op
// BenchmarkAddZones/zones-1000-12      263773656 ns/op
// BenchmarkAddZones/zones-10000-12     259945438 ns/op
//
// It is interesting to see that the batch size of 100 has worse performance
// than the batch size of 10. There is no significant difference between batch
// size of 1000 and 10000.
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
	err := batch.Finish()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zones, err := GetZones(db, nil)
		if err != nil {
			b.Fatal(err)
		}
		if len(zones) != zonesNum {
			b.Fatalf("invalid number of zones returned %d", len(zones))
		}
	}
}
