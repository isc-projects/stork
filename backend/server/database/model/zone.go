package dbmodel

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

type ZoneRelation string

const (
	ZoneRelationLocalZones    = "LocalZones"
	ZoneRelationLocalZonesApp = "LocalZones.Daemon.App"
)

// Filter used in the GetZones function for complex filtering of
// the zones returned from the database.
type GetZonesFilter struct {
	// Filter by an explicit app ID.
	AppID *int64
	// Filter by DNS app type (e.g., "bind9").
	AppType *string
	// Filter by class (typically, IN).
	Class *string
	// Filter by lower bound zone.
	LowerBound *string
	// Limit the number of zones returned.
	Limit *int
	// Paging offset.
	Offset *int
	// Filter by partial or exact zone serial.
	Serial *string
	// Filter by zone type (e.g., primary or secondary).
	Type *string
	// Filter by partial zone name, app name or view.
	Text *string
}

// Represents a zone in a database. The same zone can be shared between
// many DNS servers. Associations with different servers is are created
// by adding LocalZone instances to the zone.
type Zone struct {
	ID         int64
	Name       string
	Rname      string
	LocalZones []*LocalZone `pg:"rel:has-many"`
}

// Represents association between a server and a zone. The server
// specific zone information is held in this structure.
type LocalZone struct {
	ID       int64
	ZoneID   int64
	DaemonID int64
	View     string

	Class    string
	Serial   int64 `pg:",use_zero"`
	Type     string
	LoadedAt time.Time

	Daemon *Daemon `pg:"rel:has-one"`
	Zone   *Zone   `pg:"rel:has-one"`
}

// Upserts multiple zones in a transaction into the database.
func addZones(tx *pg.Tx, zones ...*Zone) error {
	// First insert zones into the zone table.
	_, err := tx.Model(&zones).OnConflict("(name) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("rname = EXCLUDED.rname").
		Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert %d zones into the database", len(zones))
	}
	// Next, insert all local zones .
	localZones := []*LocalZone{}
	for _, zone := range zones {
		for _, localZone := range zone.LocalZones {
			localZone.ZoneID = zone.ID
			localZones = append(localZones, localZone)
		}
	}

	_, err = tx.Model(&localZones).OnConflict("(zone_id, daemon_id, view) DO UPDATE").
		Set("class = EXCLUDED.class").
		Set("serial = EXCLUDED.serial").
		Set("type = EXCLUDED.type").
		Set("loaded_at = EXCLUDED.loaded_at").
		Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert %d local zones into the database", len(localZones))
	}
	return nil
}

// Upserts multiple zones into the database. It creates new transaction if the
// transaction has not been started yet. Otherwise, it uses an existing transaction.
func AddZones(dbi pg.DBI, zones ...*Zone) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addZones(tx, zones...)
		})
	}
	return addZones(dbi.(*pg.Tx), zones...)
}

// Retrieves a list of zones from the database with optional relations and filtering.
// The ORM-based implementation may result in multiple queries when deep relations
// (with daemon and with app) are used. The only alternative would be raw queries.
// However, raw queries don't improve performance of getting the zones for one
// relation (LocalZones). They could possibly improve the performance when cascaded
// relations (i.e., LocalZones.Daemon.App) are used. Unfortunately, it would significantly
// complicate the implementation. Note that this function is primarily used for
// paging zones, so the number of records is typically low, and the performance gain
// would be negligible.
func GetZones(db pg.DBI, filter *GetZonesFilter, relations ...ZoneRelation) ([]*Zone, int, error) {
	var zones []*Zone
	q := db.Model(&zones)
	// Add relations.
	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	// Order expression.
	q = q.OrderExpr("rname ASC")

	// Filtering is optional.
	if filter != nil {
		// Limit the number of zones returned.
		if filter.Limit != nil {
			q = q.Limit(*filter.Limit)
		}
		// Paging from the last returned zone name.
		if filter.LowerBound != nil {
			labels := dns.SplitDomainName(*filter.LowerBound)
			slices.Reverse(labels)
			lowerBound := strings.Join(labels, ".")
			q = q.Where("rname > ?", lowerBound)
		}
		// Paging from offset.
		if filter.Offset != nil {
			q = q.Offset(*filter.Offset)
		}
		// Join relations required for filtering.
		if filter.Serial != nil || filter.Class != nil || filter.Type != nil || filter.AppID != nil || filter.AppType != nil || filter.Text != nil {
			q = q.Join("JOIN local_zone AS lz").JoinOn("lz.zone_id = zone.id")
			if filter.AppID != nil || filter.AppType != nil || filter.Text != nil {
				q = q.Join("JOIN daemon AS d").JoinOn("d.id = lz.daemon_id").
					Join("JOIN app AS a").JoinOn("a.id = d.app_id")
			}
		}
		// Filter by serial.
		if filter.Serial != nil {
			q = q.Where("lz.serial::text ILIKE ?", "%"+*filter.Serial+"%")
		}
		// Filter by class.
		if filter.Class != nil {
			q = q.Where("lz.class = ?", *filter.Class)
		}
		// Filter by type.
		if filter.Type != nil {
			q = q.Where("lz.type = ?", *filter.Type)
		}
		// Filter by app ID.
		if filter.AppID != nil {
			q = q.Where("a.id = ?", *filter.AppID)
		}
		// Filter by app type.
		if filter.AppType != nil {
			q = q.Where("a.type = ?", *filter.AppType)
		}
		// Filter by zone name, app name or local zone view using partial matching.
		if filter.Text != nil {
			q = q.WhereGroup(func(q *pg.Query) (*pg.Query, error) {
				return q.WhereOr("zone.name ILIKE ?", "%"+*filter.Text+"%").
					WhereOr("a.name ILIKE ?", "%"+*filter.Text+"%").
					WhereOr("lz.view ILIKE ?", "%"+*filter.Text+"%"), nil
			})
		}
	}
	count, err := q.SelectAndCount()
	if err != nil {
		return nil, count, errors.Wrapf(err, "failed to select zones from the database")
	}
	return zones, count, nil
}

// Deletes zones which are not associated with any daemons. Returns deleted zone
// count and an error.
func DeleteOrphanedZones(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]LocalZone{}).
		Column("id").
		Limit(1).
		Where("zone.id = local_zone.zone_id")
	result, err := dbi.Model(&[]Zone{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = errors.Wrapf(err, "failed to delete orphaned zones")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Deletes associations between a daemon and the zones.
func DeleteLocalZones(db pg.DBI, daemonID int64) error {
	_, err := db.Model((*LocalZone)(nil)).Where("daemon_id = ?", daemonID).Delete()
	return errors.Wrapf(err, "failed to delete local zones for daemon id %d", daemonID)
}

// go-pg hook triggered before zone insert into the database. It sets the
// rname from name. The rname column is used for ordering the zones in DNS
// order.
func (zone *Zone) BeforeInsert(ctx context.Context) (context.Context, error) {
	zone.Rname = storkutil.ConvertNameToRname(zone.Name)
	return ctx, nil
}

// go-pg hook triggered before zone update in the database. It sets the
// rname from name. The rname column is used for ordering the zones in DNS
// order.
func (zone *Zone) BeforeUpdate(ctx context.Context) (context.Context, error) {
	zone.Rname = storkutil.ConvertNameToRname(zone.Name)
	return ctx, nil
}
