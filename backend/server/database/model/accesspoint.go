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
	DaemonID int64           `pg:",pk"`
	Daemon   *Daemon         `pg:"rel:has-one"`
	Type     AccessPointType `pg:",pk"`
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

// Add or update an access point in the database.
func addOrUpdateAccessPoint(db dbops.DBI, accessPoint *AccessPoint) error {
	// If the access point already exists, update it.
	_, err := db.Model(accessPoint).WherePK().OnConflict("(daemon_id, type) DO UPDATE").Insert()
	if err != nil {
		return pkgerrors.Wrapf(
			err,
			"problem adding or updating access point: %v",
			accessPoint,
		)
	}
	return nil
}

// Deletes all access points for a given daemon that don't match the provided
// types. If `keepTypes` is empty, all access points for the daemon will be
// deleted.
func deleteAccessPointsExcept(db dbops.DBI, daemonID int64, keepTypes []AccessPointType) error {
	accessPoint := &AccessPoint{DaemonID: daemonID}
	query := db.Model(accessPoint).Where("daemon_id = ?", daemonID)

	if len(keepTypes) > 0 {
		query.Where("type NOT IN (?)", pg.In(keepTypes))
	}

	_, err := query.Delete()
	if err != nil {
		return pkgerrors.Wrapf(
			err,
			"problem deleting access points for daemon: %d, keeping types: %v",
			daemonID,
			keepTypes,
		)
	}
	return nil
}
