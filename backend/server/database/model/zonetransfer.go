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
// and started_at. Therefore, these fields must not be NULL, and for optional fields
// use_zero tag must be used to avoid NOT NULL constraint violation.
type ZoneTransferState struct {
	ID                int64
	DaemonID          int64
	CreatedAt         time.Time
	ViewName          string `pg:",use_zero"`
	ZoneName          string
	Serial            *int64 `pg:",use_zero"`
	Client            string `pg:",use_zero"`
	Server            string
	MessagesCount     int64
	RecordsCount      int64
	BytesCount        int64
	BytesPerSecond    int64
	Duration          time.Duration
	EffectiveDuration time.Duration `pg:"-"`
	Status            bind9xfr.Status
	StartedAt         time.Time `pg:",use_zero"`
	CompletedAt       time.Time
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
	statuses map[bind9xfr.Status]bool
}

// Instantiates the statuses filter.
func NewGetZoneTransferStatesStatuses() *GetZoneTransferStatesStatuses {
	return &GetZoneTransferStatesStatuses{
		statuses: make(map[bind9xfr.Status]bool),
	}
}

// Enables a filter for a specific status.
func (f *GetZoneTransferStatesStatuses) Enable(status bind9xfr.Status) {
	f.statuses[status] = true
}

// Returns an iterator over the enabled statuses.
func (f *GetZoneTransferStatesStatuses) GetEnabled() iter.Seq[bind9xfr.Status] {
	return func(yield func(bind9xfr.Status) bool) {
		for status, enabled := range f.statuses {
			if enabled {
				if !yield(status) {
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
	// Paging offset.
	Offset *int
	// Filter by partial zone serial number.
	Serial *string
	// Filter by multiple statuses of the zone transfers.
	Statuses *GetZoneTransferStatesStatuses
	// Filter by partial zone name, view name, client name,
	// server name, or message text.
	Text *string
	// Filter by ID of the machine where the server for that zone
	// transfer is running.
	ServerMachineID *int64
	// Filter by ID of the machine where the client for that zone
	// transfer is running.
	ClientMachineID *int64
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
// and started_at already exists. The common use case is when the started zone transfer
// was recorded in the database, and it subsequently ended. In this case, we must
// mark it completed, and update the related statistics.
func addOrUpdateZoneTransferState(dbi pg.DBI, zoneTransferState *ZoneTransferState) error {
	_, err := dbi.Model(zoneTransferState).
		OnConflict("(daemon_id, view_name, zone_name, client, started_at) DO UPDATE").
		Set("serial = EXCLUDED.serial").
		Set("server = EXCLUDED.server").
		Set("messages_count = EXCLUDED.messages_count").
		Set("records_count = EXCLUDED.records_count").
		Set("bytes_count = EXCLUDED.bytes_count").
		Set("bytes_per_second = EXCLUDED.bytes_per_second").
		Set("duration = EXCLUDED.duration").
		Set("status = EXCLUDED.status").
		Set("completed_at = EXCLUDED.completed_at").
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
// and started_at already exists. The common use case is when the started zone transfer
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
	// In this case, we can calculate the duration by subtracting the started_at
	// from the current time, if the started_at is set. This expression is used
	// to conditionally calculate the duration.
	const effectiveDurationExpr = `
	CASE
		WHEN COALESCE(zone_transfer_state.duration, 0) = 0
			AND zone_transfer_state.started_at > '1970-01-01'
		THEN (
			EXTRACT(
				EPOCH FROM (now() at time zone 'utc' - zone_transfer_state.started_at)
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
		Column("zone_transfer_state.started_at").
		Column("zone_transfer_state.serial").
		Column("zone_transfer_state.server").
		Column("zone_transfer_state.messages_count").
		Column("zone_transfer_state.records_count").
		Column("zone_transfer_state.bytes_count").
		Column("zone_transfer_state.bytes_per_second").
		Column("zone_transfer_state.duration").
		ColumnExpr(effectiveDurationExpr + " AS effective_duration").
		Column("zone_transfer_state.status").
		Column("zone_transfer_state.completed_at").
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

	// Filter by the ID of the machine where the XFR server is running.
	if filter.ServerMachineID != nil {
		q = q.Where("zone_transfer_state.server_machine_id = ?", *filter.ServerMachineID)
	}

	// Filter by the ID of the machine where the XFR client is running.
	if filter.ClientMachineID != nil {
		q = q.Where("zone_transfer_state.client_machine_id = ?", *filter.ClientMachineID)
	}

	// Filter by zone name, daemon name or local zone view using partial matching.
	if filter.Text != nil {
		// Ensure case-insensitive comparison against root and (root).
		filterText := strings.ToLower(*filter.Text)
		q = q.WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			// UI can use the keyword "root" or "(root)" to search for transfers pertaining
			// to the root zone. That's because the root zone is displayed using the keywords
			// in the UI. Users will expect that the root zone transfers are returned not only
			// when they type the dot but also the keyword.
			//nolint:gocritic
			if strings.HasPrefix("root", filterText) || strings.HasPrefix("(root)", filterText) {
				q = q.Where("zone_transfer_state.zone_name = ?", ".")
			}
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
