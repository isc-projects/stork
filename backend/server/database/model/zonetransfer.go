package dbmodel

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"isc.org/stork/daemondata/bind9xfr"
)

// Represents a relations between the zone_transfer_state and other tables.
type ZoneTransferStateRelation string

const (
	// Relation to the machine where XFR client is running.
	ZoneTransferStateRelationClientMachine ZoneTransferStateRelation = "ClientMachine"
	// Relation to the machine where XFR server is running.
	ZoneTransferStateRelationServerMachine ZoneTransferStateRelation = "ServerMachine"
)

// It represents a zone transfer state in the database. It holds the information
// captured from the BIND 9 server by the zone transfer tracker, and the association
// with the BIND 9 daemon where is information was captured.
// The zone transfer state is inserted with ON CONFLICT DO UPDATE clause, and the
// conflict is checked for the following fields: daemon_id, view_name, zone_name, client,
// and start_time. Therefore, these fields must not be NULL, and for optional fields
// use_zero tag must be used to avoid NOT NULL constraint violation.
type ZoneTransferState struct {
	ID                int64
	DaemonID          int64
	CreatedAt         time.Time
	ViewName          string `pg:",use_zero"`
	ZoneName          string
	Serial            int64
	Client            string `pg:",use_zero"`
	Server            string
	MessagesCount     int64
	RecordsCount      int64
	BytesCount        int64
	Duration          time.Duration
	EffectiveDuration time.Duration `pg:"-"`
	Status            bind9xfr.Status
	StartTime         time.Time `pg:",use_zero"`
	CompletionTime    time.Time
	Message           string
	Local             bool `pg:",use_zero"`
	ClientMachineID   int64
	ServerMachineID   int64

	ClientMachine *Machine `pg:"rel:has-one"`
	ServerMachine *Machine `pg:"rel:has-one"`
}

// Holds a set of zone transfer statuses by which the zone transfers should be filtered.
// If there are no statuses specified, all zone transfers are returned.
// Otherwise, the zone transfers matching the enabled filters are returned.
type GetZoneTransferStatesStatuses struct {
	types map[bind9xfr.Status]bool
}

// Instantiates the zone types filter.
func NewGetZoneTransferStatesStatuses() *GetZoneTransferStatesStatuses {
	return &GetZoneTransferStatesStatuses{
		types: make(map[bind9xfr.Status]bool),
	}
}

// Enables a filter for a specific zone type. The zones of the matching
// type are returned.
func (f *GetZoneTransferStatesStatuses) Enable(status bind9xfr.Status) {
	f.types[status] = true
}

// Returns an iterator over the enabled zone types.
// Since primary is an alias of master, and the secondary is an alias of slave,
// the iterator includes both primary and master, and/or secondary and slave,
// if one in any pair is enabled. The GetZonesFilter.EnableZoneType() function
// includes a special logic to enable both aliases if one of them is enabled.
func (f *GetZoneTransferStatesStatuses) GetEnabled() iter.Seq[bind9xfr.Status] {
	return func(yield func(bind9xfr.Status) bool) {
		for zoneType, enabled := range f.types {
			if enabled {
				if !yield(zoneType) {
					return
				}
			}
		}
	}
}

// Filter used in the GetZoneTransferStates function for complex filtering of
// the zone transfer states returned from the database.
type GetZoneTransferStatesFilter struct {
	// Limit the number of zone transfer states returned.
	Limit *int
	// Filter by ID of the machine where the primary or secondary
	// server for that zone transfer is running.
	MachineID *int64
	// Paging offset.
	Offset *int
	// Filter by partial zone serial number.
	Serial *string
	// Filter by multiple statuses of the zone transfers.
	Statuses *GetZoneTransferStatesStatuses
	// Filter by partial zone name, view name, client name,
	// server name, or message text.
	Text *string
	// Exclude local zone transfers (i.e., transfers initiated by the client
	// running on the same machine as the server) in the results. It would
	// exclude the transfers initiated by Stork.
	ExcludeLocal bool
}

// Convenience function to enable a zone transfer status filter.
func (f *GetZoneTransferStatesFilter) EnableStatus(status bind9xfr.Status) {
	if f.Statuses == nil {
		f.Statuses = NewGetZoneTransferStatesStatuses()
	}
	f.Statuses.Enable(status)
}

// Adds a zone transfer state into the database. It updates the existing record
// if the zone transfer state with the same daemon_id, view_name, zone_name, client,
// and start_time already exists. The common use case is when the started zone transfer
// was recorded in the database, and it subsequently ended. In this case, we must
// mark it completed, and update the related statistics.
func addOrUpdateZoneTransferState(dbi pg.DBI, zoneTransferState *ZoneTransferState) error {
	_, err := dbi.Model(zoneTransferState).
		OnConflict("(daemon_id, view_name, zone_name, client, start_time) DO UPDATE").
		Set("serial = EXCLUDED.serial").
		Set("server = EXCLUDED.server").
		Set("messages_count = EXCLUDED.messages_count").
		Set("records_count = EXCLUDED.records_count").
		Set("bytes_count = EXCLUDED.bytes_count").
		Set("duration = EXCLUDED.duration").
		Set("status = EXCLUDED.status").
		Set("completion_time = EXCLUDED.completion_time").
		Set("message = EXCLUDED.message").
		Set("local = EXCLUDED.local").
		Set("client_machine_id = EXCLUDED.client_machine_id").
		Set("server_machine_id = EXCLUDED.server_machine_id").
		Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert zone transfer state for zone %s, view %s, daemon %d into the database", zoneTransferState.ZoneName, zoneTransferState.ViewName, zoneTransferState.DaemonID)
	}
	return nil
}

// Adds a zone transfer state into the database. It updates the existing record
// if the zone transfer state with the same daemon_id, view_name, zone_name, client,
// and start_time already exists. The common use case is when the started zone transfer
// was recorded in the database, and it subsequently ended. In this case, we must
// mark it completed, and update the related statistics. The function creates a new
// transaction if the database is not already in a transaction. Otherwise, it uses
// the existing transaction.
func AddOrUpdateZoneTransferState(dbi pg.DBI, zoneTransferState *ZoneTransferState) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addOrUpdateZoneTransferState(tx, zoneTransferState)
		})
	}
	return addOrUpdateZoneTransferState(dbi.(*pg.Tx), zoneTransferState)
}

// Returns a page of zone transfer states from the database with optional filtering
// and relations. If no filtering is specified, all zone transfer states are returned.
// The optional relations can be used to join the machine table by the
// client and server machine IDs. The joined machine table only contains the id, address
// and agent port fields.Other fields are excluded for performance reasons.
func GetZoneTransferStatesByPage(dbi pg.DBI, filter *GetZoneTransferStatesFilter, sortField string, sortDir SortDirEnum, getZoneTransferStatesRelations ...ZoneTransferStateRelation) ([]*ZoneTransferState, int64, error) {
	var zoneTransfers []*ZoneTransferState
	// The duration is NULL when the zone transfer has not yet completed.
	// In this case, we can calculate the duration by subtracting the start_time
	// from the current time, if the start_time is set. This expression is used
	// to conditionally calculate the duration.
	const effectiveDurationExpr = `
	CASE
		WHEN COALESCE(zone_transfer_state.duration, 0) = 0
			AND zone_transfer_state.start_time > '1970-01-01'
		THEN (
			EXTRACT(
				EPOCH FROM (now() at time zone 'utc' - zone_transfer_state.start_time)
			) * 1000000000)::bigint
		ELSE zone_transfer_state.duration
	END`

	q := dbi.Model(&zoneTransfers).
		Column("zone_transfer_state.id").
		Column("zone_transfer_state.created_at").
		Column("zone_transfer_state.daemon_id").
		Column("zone_transfer_state.view_name").
		Column("zone_transfer_state.zone_name").
		Column("zone_transfer_state.client").
		Column("zone_transfer_state.start_time").
		Column("zone_transfer_state.serial").
		Column("zone_transfer_state.server").
		Column("zone_transfer_state.messages_count").
		Column("zone_transfer_state.records_count").
		Column("zone_transfer_state.bytes_count").
		Column("zone_transfer_state.duration").
		ColumnExpr(effectiveDurationExpr + " AS effective_duration").
		Column("zone_transfer_state.status").
		Column("zone_transfer_state.completion_time").
		Column("zone_transfer_state.message").
		Column("zone_transfer_state.local").
		Column("zone_transfer_state.client_machine_id").
		Column("zone_transfer_state.server_machine_id")

	for _, relation := range getZoneTransferStatesRelations {
		// Optionally join the machine table to extract the machine
		// address where the XFR client and/or server are running.
		// Note that they can be nil if the client_machine_id or
		// server_machine_id are nil.
		q = q.Relation(fmt.Sprintf("%s.id", relation)).
			Relation(fmt.Sprintf("%s.address", relation)).
			Relation(fmt.Sprintf("%s.agent_port", relation))
	}

	orderExpr, _ := prepareOrderAndDistinctExpr("zone_transfer_state", sortField, sortDir, func(sortField string, escapedTableName string, dirExpr string) (string, string, bool) {
		if sortField == "effective_duration" {
			// Effective duration is not a column in the database but an expression.
			// We must use custom handler to avoid treating it as a column name.
			return sortField + " " + dirExpr, sortField, true
		}
		return "", "", false
	})
	q = q.OrderExpr(orderExpr)

	// Filtering is optional.
	if filter == nil {
		total, err := q.SelectAndCount()
		if err != nil && !errors.Is(err, pg.ErrNoRows) {
			return nil, 0, errors.Wrapf(err, "failed to select zone transfer states from the database")
		}
		return zoneTransfers, int64(total), err
	}

	// Paging from offset.
	if filter.Offset != nil {
		q = q.Offset(*filter.Offset)
	}

	// Limit the number of zone transfer states returned.
	if filter.Limit != nil {
		q = q.Limit(*filter.Limit)
	}

	// Filter by partial zone serial number.
	if filter.Serial != nil {
		q = q.Where("zone_transfer_state.serial::text ILIKE ?", "%"+*filter.Serial+"%")
	}

	// Filter by statuses.
	if filter.Statuses != nil {
		statuses := slices.Collect(filter.Statuses.GetEnabled())
		if len(statuses) > 0 {
			q = q.WhereIn("zone_transfer_state.status IN (?)", statuses)
		}
	}

	// Filter by the ID of the machines where the primary or secondary
	// DNS server is running.
	if filter.MachineID != nil {
		q = q.WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			q = q.WhereOr("zone_transfer_state.client_machine_id = ?", *filter.MachineID).
				WhereOr("zone_transfer_state.server_machine_id = ?", *filter.MachineID)
			return q, nil
		})
	}

	// Filter by zone name, daemon name or local zone view using partial matching.
	if filter.Text != nil {
		// Ensure case-insensitive comparison against root and (root).
		filterText := strings.ToLower(*filter.Text)
		q = q.WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			q = q.WhereOr("zone_transfer_state.zone_name ILIKE ?", "%"+filterText+"%").
				WhereOr("zone_transfer_state.view_name ILIKE ?", "%"+*filter.Text+"%").
				WhereOr("zone_transfer_state.client ILIKE ?", "%"+*filter.Text+"%").
				WhereOr("zone_transfer_state.server ILIKE ?", "%"+*filter.Text+"%").
				WhereOr("zone_transfer_state.message ILIKE ?", "%"+*filter.Text+"%")
			return q, nil
		})
	}

	if filter.ExcludeLocal {
		// Exclude local zone transfers (e.g., transfers initiated by Stork).
		q = q.Where("zone_transfer_state.local = ?", false)
	}

	total, err := q.SelectAndCount()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		return nil, 0, errors.Wrapf(err, "failed to select zone transfer states from the database")
	}
	return zoneTransfers, int64(total), err
}
