package dbmodel

import (
	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	"isc.org/stork/datamodel/protocoltype"
	dbops "isc.org/stork/server/database"
)

// Valid kinds of the access points.
type AccessPointType string

const (
	AccessPointControl    AccessPointType = "control"
	AccessPointStatistics AccessPointType = "statistics"
)

// A structure reflecting the access_point SQL table.
type AccessPoint struct {
	ID       int64 `pg:",pk"`
	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
	Type     AccessPointType
	Address  string
	Port     int64
	// For BIND 9 when the RNDC key is set, this value is: RNDC key name,
	// algorithm and secret joined by colon.
	// For Kea when the Basic Auth is set, this is a username of the user used
	// by the Stork agent to authenticate to the Kea server.
	// Otherwise it is empty string.
	Key      string
	Protocol protocoltype.ProtocolType `pg:",use_zero"`
}

// Adds an access point in the database.
func addAccessPoint(db dbops.DBI, accessPoint *AccessPoint) error {
	_, err := db.Model(accessPoint).Insert()
	if err != nil {
		return pkgerrors.Wrapf(
			err,
			"problem adding access point: %v",
			accessPoint,
		)
	}
	return nil
}

// Updates an access point in the database.
func updateAccessPoint(db dbops.DBI, accessPoint *AccessPoint) error {
	if accessPoint.ID == 0 {
		return pkgerrors.Errorf(
			"cannot update access point without ID: %v",
			accessPoint,
		)
	}
	// If the access point doesn't exist, this will return an error.

	// If the access point already exists, update it.
	_, err := db.Model(accessPoint).WherePK().Update()
	if err != nil {
		return pkgerrors.Wrapf(
			err,
			"problem updating access point: %v",
			accessPoint,
		)
	}
	return nil
}

// Deletes all access points for a given daemon that don't match the provided
// IDs. If `keepIDs` is empty, all access points for the daemon will be
// deleted.
func deleteAccessPointsExcept(db dbops.DBI, daemonID int64, keepIDs []int64) error {
	accessPoint := &AccessPoint{DaemonID: daemonID}
	query := db.Model(accessPoint).Where("daemon_id = ?", daemonID)

	if len(keepIDs) > 0 {
		query.Where("id NOT IN (?)", pg.In(keepIDs))
	}

	_, err := query.Delete()
	if err != nil {
		return pkgerrors.Wrapf(
			err,
			"problem deleting access points for daemon: %d, keeping IDs: %v",
			daemonID,
			keepIDs,
		)
	}
	return nil
}
