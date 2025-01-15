package dbmodel

import (
	"github.com/go-pg/pg/v10"
)

type BatchUpsert[T any] struct {
	db        pg.DBI
	items     *[]T
	maxLength int
	upsertFn  func(pg.DBI, ...T) error
}

func NewBatchUpsert[T any](db pg.DBI, maxLength int, upsertFn func(pg.DBI, ...T) error) (batch *BatchUpsert[T]) {
	return &BatchUpsert[T]{
		db:        db,
		items:     &[]T{},
		maxLength: maxLength,
		upsertFn:  upsertFn,
	}
}

func (buffer *BatchUpsert[T]) Add(item T, done bool) error {
	*buffer.items = append(*buffer.items, item)
	if (done || len(*buffer.items) >= buffer.maxLength) && len(*buffer.items) > 0 {
		if err := buffer.upsertFn(buffer.db, *buffer.items...); err != nil {
			return err
		}
		*buffer.items = []T{}
	}
	return nil
}

type Zone struct {
	ID         int64
	Name       string
	LocalZones []*LocalZone `pg:"rel:has-many"`
}

type LocalZone struct {
	ID       int64
	ZoneID   int64
	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
	Zone     *Zone   `pg:"rel:has-one"`
}

type FlatZone struct {
	ID       int64
	Name     string
	ZoneID   int64
	DaemonID int64
}

func AddZone(dbi pg.DBI, zone *Zone) error {
	_, err := dbi.Model(zone).Insert()
	if err != nil {
		return err
	}
	for _, localZone := range zone.LocalZones {
		localZone.ZoneID = zone.ID
		_, err = dbi.Model(localZone).Insert()
		if err != nil {
			return err
		}
	}
	return nil
}

func AddZones(dbi pg.DBI, zones ...*Zone) error {
	_, err := dbi.Model(&zones).Insert()
	if err != nil {
		return err
	}
	localZones := []*LocalZone{}
	for _, zone := range zones {
		for _, localZone := range zone.LocalZones {
			localZone.ZoneID = zone.ID
			localZones = append(localZones, localZone)
		}
	}
	_, err = dbi.Model(&localZones).Insert()
	if err != nil {
		return err
	}

	return nil
}

func GetZones(db pg.DBI) ([]*Zone, error) {
	//	var flatZones []*FlatZone
	var zones []*Zone
	//_, err := db.Query(&flatZones, "SELECT z.id, z.name, d.id as daemon_id, lz.zone_id as zone_id FROM zone AS z INNER JOIN local_zone AS lz ON z.id = lz.zone_id INNER JOIN daemon AS d ON d.id = lz.daemon_id")
	err := db.Model(&zones).Relation("LocalZones").Select()
	if err != nil {
		return nil, err
	}
	/*	for _, fz := range flatZones {
		zone := &Zone{
			Name: fz.Name,
		}
		zones = append(zones, zone)
	} */

	return zones, nil
}
