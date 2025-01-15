package dbmodel

import (
	"context"
	"testing"

	"github.com/go-pg/pg/v10"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

func BenchmarkAddZones(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	randomZones := testutil.GenerateRandomZones(100000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			machine := &Machine{
				ID:        0,
				Address:   "localhost",
				AgentPort: int64(8080 + i),
			}
			err := AddMachine(tx, machine)
			if err != nil {
				return err
			}

			app := &App{
				ID:        0,
				MachineID: machine.ID,
				Type:      AppTypeKea,
				Daemons: []*Daemon{
					NewBind9Daemon(true),
				},
			}
			addedDaemons, err := AddApp(tx, app)
			if err != nil {
				return err
			}

			zone := &Zone{
				Name: randomZones[i].Name,
				LocalZones: []*LocalZone{
					{
						DaemonID: addedDaemons[0].ID,
					},
				},
			}
			err = AddZone(tx, zone)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetZones(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	zonesNum := 1000000
	randomZones := testutil.GenerateRandomZones(zonesNum)

	var daemons []*Daemon
	for i := range randomZones {
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

	batch := NewBatchUpsert(db, 10000, func(db pg.DBI, zones ...*Zone) error {
		return AddZones(db, zones...)
	})
	for i, randomZone := range randomZones {
		zone := &Zone{
			Name: randomZone.Name,
			LocalZones: []*LocalZone{
				{
					DaemonID: daemons[i/(zonesNum/1000)].ID,
				},
			},
		}
		err := batch.Add(zone, i+1 == len(randomZones))
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zones, err := GetZones(db)
		if err != nil {
			b.Fatal(err)
		}
		if len(zones) != zonesNum {
			b.Fatalf("invalid number of zones returned %d", len(zones))
		}
	}
}
