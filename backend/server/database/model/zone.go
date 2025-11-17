package dbmodel

import (
	"context"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

type ZoneRelation string

const (
	ZoneRelationLocalZones             ZoneRelation = "LocalZones"
	ZoneRelationLocalZonesDaemon       ZoneRelation = "LocalZones.Daemon"
	ZoneRelationLocalZonesAccessPoints ZoneRelation = "LocalZones.Daemon.AccessPoints"
	ZoneRelationLocalZonesMachine      ZoneRelation = "LocalZones.Daemon.Machine"
)

type ZoneType string

const (
	ZoneTypeBuiltin        ZoneType = "builtin"
	ZoneTypeConsumer       ZoneType = "consumer"
	ZoneTypeDelegationOnly ZoneType = "delegation-only"
	ZoneTypeForward        ZoneType = "forward"
	ZoneTypeHint           ZoneType = "hint"
	ZoneTypeMaster         ZoneType = "master"
	ZoneTypeMirror         ZoneType = "mirror"
	ZoneTypeNative         ZoneType = "native"
	ZoneTypePrimary        ZoneType = "primary"
	ZoneTypeProducer       ZoneType = "producer"
	ZoneTypeRedirect       ZoneType = "redirect"
	ZoneTypeSecondary      ZoneType = "secondary"
	ZoneTypeSlave          ZoneType = "slave"
	ZoneTypeStaticStub     ZoneType = "static-stub"
	ZoneTypeStub           ZoneType = "stub"
)

// Holds a set of zone types by which the zones should be filtered.
// If there are no types specified, all zones are returned.
// Otherwise, the zones matching the enabled filters are returned.
type GetZonesFilterZoneTypes struct {
	types map[ZoneType]bool
}

// Instantiates the zone types filter.
func NewGetZonesFilterZoneTypes() *GetZonesFilterZoneTypes {
	return &GetZonesFilterZoneTypes{
		types: make(map[ZoneType]bool),
	}
}

// Enables a filter for a specific zone type. The zones of the matching
// type are returned.
func (f *GetZonesFilterZoneTypes) Enable(zoneType ZoneType) {
	f.types[zoneType] = true
}

// Returns true if any filter is specified (enabled or disabled).
func (f *GetZonesFilterZoneTypes) IsAnySpecified() bool {
	return len(f.types) > 0
}

// Returns an iterator over the enabled zone types.
// Since primary is an alias of master, and the secondary is an alias of slave,
// the iterator includes both primary and master, and/or secondary and slave,
// if one in any pair is enabled. The GetZonesFilter.EnableZoneType() function
// includes a special logic to enable both aliases if one of them is enabled.
func (f *GetZonesFilterZoneTypes) GetEnabled() iter.Seq[ZoneType] {
	return func(yield func(ZoneType) bool) {
		for zoneType, enabled := range f.types {
			if enabled {
				if !yield(zoneType) {
					return
				}
			}
		}
	}
}

// Filter used in the GetZones function for complex filtering of
// the zones returned from the database.
type GetZonesFilter struct {
	// Filter by an explicit daemon ID.
	DaemonID *int64
	// Filter by DNS daemon name (e.g., "bind9").
	DaemonName *daemonname.Name
	// Filter by machine ID.
	// TODO: Code implemented in below line is a temporary solution for virtual applications.
	MachineID *int64
	// Filter by class (typically, IN).
	Class *string
	// Filter by lower bound zone.
	LowerBound *string
	// Limit the number of zones returned.
	Limit *int
	// Paging offset.
	Offset *int
	// Filter by response policy zone. If nil, all zones are returned.
	RPZ *bool
	// Filter by partial or exact zone serial.
	Serial *string
	// Filter by zone type (e.g., primary or secondary).
	Types *GetZonesFilterZoneTypes
	// Filter by partial zone name, daemon name or view.
	Text *string
}

// Convenience function to enable a zone type filter. There are two zone types that
// have aliases: primary and master, and secondary and slave. The function enables
// filters for both aliases if one of them is enabled.
func (f *GetZonesFilter) EnableZoneType(zoneType ZoneType) {
	if f.Types == nil {
		f.Types = NewGetZonesFilterZoneTypes()
	}
	switch zoneType {
	case ZoneTypePrimary, ZoneTypeMaster:
		f.Types.Enable(ZoneTypePrimary)
		f.Types.Enable(ZoneTypeMaster)
	case ZoneTypeSecondary, ZoneTypeSlave:
		f.Types.Enable(ZoneTypeSecondary)
		f.Types.Enable(ZoneTypeSlave)
	default:
		f.Types.Enable(zoneType)
	}
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

// Returns a local zone for a specific daemon and view.
func (zone *Zone) GetLocalZone(daemonID int64, view string) *LocalZone {
	for _, localZone := range zone.LocalZones {
		if localZone.DaemonID == daemonID && localZone.View == view {
			return localZone
		}
	}
	return nil
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
	RPZ      bool
	LoadedAt time.Time

	Daemon *Daemon `pg:"rel:has-one"`
	Zone   *Zone   `pg:"rel:has-one"`

	ZoneTransferAt *time.Time `pg:"zone_transfer_at"`
}

// Represents the counts of zones returned by the GetZoneCountStatsByDaemon.
type ZoneCountStats struct {
	DistinctZones int64 `pg:"distinct_zones"`
	BuiltinZones  int64 `pg:"builtin_zones"`
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

// Sort field which may be used in GetZones.
// If any of these fields is used, it means that the sorting must be done
// based on a field of the related table. The "zone" table needs to be
// JOINed first with other relation table. The relation is often of
// "has-many" type (e.g. one zone may have many local_zones),
// so after such JOIN the results will no longer have distinct zone IDs.
// In order to have distinct IDs, a subquery is used in the JOIN operation, which
// aggregates only one target relation record per zone ID.
type ZoneSortField string

// Valid sort fields.
const (
	LocalZoneSerial ZoneSortField = "distinct_lz.serial"
	LocalZoneType   ZoneSortField = "distinct_lz.type"
)

// Retrieves a list of zones from the database with optional relations and filtering.
// The ORM-based implementation may result in multiple queries when deep relations
// (with daemon) are used. The only alternative would be raw queries.
// However, raw queries don't improve performance of getting the zones for one
// relation (LocalZones). They could possibly improve the performance when cascaded
// relations (i.e., LocalZones.Daemon) are used. Unfortunately, it would significantly
// complicate the implementation. Note that this function is primarily used for
// paging zones, so the number of records is typically low, and the performance gain
// would be negligible.
func GetZones(db pg.DBI, filter *GetZonesFilter, sortField string, sortDir SortDirEnum, relations ...ZoneRelation) ([]*Zone, int, error) { //nolint: gocyclo
	var zones []*Zone
	q := db.Model(&zones).Group("zone.id")
	// Add relations.
	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	// Order expression.
	orderExpr, _ := prepareOrderExpr("zone", sortField, sortDir)
	q = q.OrderExpr(orderExpr)
	if ZoneSortField(sortField) == LocalZoneSerial {
		sortSubquery := db.Model((*LocalZone)(nil)).
			Column("zone_id").
			ColumnExpr("MIN(serial) AS serial").
			Group("zone_id")
		q = q.Join("LEFT JOIN (?) AS distinct_lz", sortSubquery).JoinOn("zone.id = distinct_lz.zone_id")
		q = q.Group("distinct_lz.serial")
	}
	if ZoneSortField(sortField) == LocalZoneType {
		sortSubquery := db.Model((*LocalZone)(nil)).
			Column("zone_id").
			ColumnExpr("MIN(type) AS type").
			Group("zone_id")
		q = q.Join("LEFT JOIN (?) AS distinct_lz", sortSubquery).JoinOn("zone.id = distinct_lz.zone_id")
		q = q.Group("distinct_lz.type")
	}

	// Filtering is optional.
	if filter == nil {
		count, err := q.SelectAndCount()
		if err != nil {
			return nil, count, errors.Wrapf(err, "failed to select unfiltered zones from the database")
		}
		return zones, count, nil
	}

	// Limit the number of zones returned.
	if filter.Limit != nil {
		q = q.Limit(*filter.Limit)
	}
	// Paging from the last returned zone name.
	if filter.LowerBound != nil {
		labels := dns.SplitDomainName(*filter.LowerBound)
		slices.Reverse(labels)
		lowerBound := strings.Join(labels, ".")
		q = q.Where("rname COLLATE \"C\" > ?", lowerBound)
	}
	// Paging from offset.
	if filter.Offset != nil {
		q = q.Offset(*filter.Offset)
	}
	// Join relations required for filtering.
	if filter.Serial != nil || filter.Class != nil ||
		filter.Types != nil && filter.Types.IsAnySpecified() ||
		filter.RPZ != nil || filter.DaemonID != nil ||
		filter.DaemonName != nil || filter.Text != nil ||
		// TODO: Code implemented in below line is a temporary solution for virtual applications.
		filter.MachineID != nil {
		q = q.Join("JOIN local_zone AS lz").JoinOn("lz.zone_id = zone.id")
		if filter.DaemonName != nil || filter.Text != nil ||
			// TODO: Code implemented in below line is a temporary solution for virtual applications.
			filter.MachineID != nil {
			q = q.Join("JOIN daemon AS d").JoinOn("d.id = lz.daemon_id")
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
	// Filter by zone types.
	if filter.Types != nil {
		types := slices.Collect(filter.Types.GetEnabled())
		if len(types) > 0 {
			q = q.WhereIn("lz.type IN (?)", types)
		}
	}
	// Filter by response policy zone.
	if filter.RPZ != nil {
		q = q.Where("lz.rpz = ?", *filter.RPZ)
	}
	// Filter by daemon ID.
	if filter.DaemonID != nil {
		q = q.Where("lz.daemon_id = ?", *filter.DaemonID)
	}
	// Filter by daemon name.
	if filter.DaemonName != nil {
		q = q.Where("d.name ILIKE ?", "%"+*filter.DaemonName+"%")
	}
	// Filter by machine ID.
	// TODO: Code implemented in below block is a temporary solution for virtual applications.
	if filter.MachineID != nil {
		q = q.Where("d.machine_id = ?", *filter.MachineID)
	}
	// Filter by zone name, daemon name or local zone view using partial matching.
	if filter.Text != nil {
		// Ensure case-insensitive comparison against root and (root).
		filterText := strings.ToLower(*filter.Text)
		q = q.WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			// UI can use the keyword "root" or "(root)" to search for a root zone.
			// That's because the root zone is displayed using the keywords in the UI.
			// Users will expect that the root zone is returned not only when they type
			// the dot but also the keyword.
			//nolint:gocritic
			if strings.HasPrefix("root", filterText) || strings.HasPrefix("(root)", filterText) {
				q = q.Where("zone.name = ?", ".")
			}
			q = q.WhereOr("zone.name ILIKE ?", "%"+filterText+"%").
				WhereOr("lz.view ILIKE ?", "%"+*filter.Text+"%")
			return q, nil
		})
	}
	// Select and count the zones.
	count, err := q.SelectAndCount()
	if err != nil {
		return nil, count, errors.Wrapf(err, "failed to select filtered zones from the database")
	}
	return zones, count, nil
}

// GetZoneCountStatsByDaemon returns the count of distinct zones and builtin zones for a specific daemon.
func GetZoneCountStatsByDaemon(db pg.DBI, daemonID int64) (*ZoneCountStats, error) {
	var stats ZoneCountStats
	_, err := db.QueryOne(&stats, `
		SELECT
			COUNT(DISTINCT zone_id) as distinct_zones,
			COUNT(DISTINCT zone_id) FILTER (WHERE type='builtin') as builtin_zones
		FROM local_zone
		WHERE daemon_id = ?
	`, daemonID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get zone count stats for daemon id %d", daemonID)
	}
	return &stats, nil
}

// Retrieves a zone with optional relations by its ID.
func GetZoneByID(db pg.DBI, id int64, relations ...ZoneRelation) (*Zone, error) {
	var zone Zone
	q := db.Model(&zone)
	// Add relations.
	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	q = q.Where("id = ?", id)
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to select zone with the ID of %d", id)
	}
	return &zone, nil
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

// Updates timestamp of the last RRs fetch for a local zone.
func UpdateLocalZoneRRsTransferAt(db pg.DBI, localZoneID int64) error {
	_, err := db.Model((*LocalZone)(nil)).
		Column("zone_transfer_at").
		Set("zone_transfer_at = ?", time.Now().UTC()).
		Where("id = ?", localZoneID).
		Update()
	return errors.Wrapf(err, "failed to update RRs transfer time for local zone id %d", localZoneID)
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
