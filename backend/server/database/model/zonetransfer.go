package dbmodel

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"isc.org/stork/daemondata/bind9xfr"
)

// It represents a zone transfer state in the database. It holds the information
// captured from the BIND 9 server by the zone transfer tracker, and the association
// with the BIND 9 daemon where is information was captured.
// The zone transfer state is inserted with ON CONFLICT DO UPDATE clause, and the
// conflict is checked for the following fields: daemon_id, view_name, zone_name, client,
// and start_time. Therefore, these fields must not be NULL, and for optional fields
// use_zero tag must be used to avoid NOT NULL constraint violation.
type ZoneTransferState struct {
	ID              int64
	DaemonID        int64
	CreatedAt       time.Time
	ViewName        string `pg:",use_zero"`
	ZoneName        string
	Serial          int64
	Client          string `pg:",use_zero"`
	Server          string
	MessagesCount   int64
	RecordsCount    int64
	BytesCount      int64
	Duration        time.Duration
	Status          bind9xfr.Status
	StartTime       time.Time `pg:",use_zero"`
	CompletionTime  time.Time
	Message         string
	ClientMachineID int64
	ServerMachineID int64
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
func AddorUpdateZoneTransferState(dbi pg.DBI, zoneTransferState *ZoneTransferState) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addOrUpdateZoneTransferState(tx, zoneTransferState)
		})
	}
	return addOrUpdateZoneTransferState(dbi.(*pg.Tx), zoneTransferState)
}

// Returns a page of zone transfer states from the database. The returned records
// are sorted by the creation time in descending order.
func GetZoneTransferStatesByPage(dbi pg.DBI, offset, limit int64) ([]*ZoneTransferState, int64, error) {
	var zoneTransfers []*ZoneTransferState
	q := dbi.Model(&zoneTransfers).
		Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(limit))
	total, err := q.SelectAndCount()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		return nil, 0, errors.Wrapf(err, "failed to select zone transfer states from the database")
	}
	return zoneTransfers, int64(total), err
}
