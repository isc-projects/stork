package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
)

// A structure reflecting the access_point SQL table.
type AccessPoint struct {
	AppID     int64  `pg:",pk"`
	Type      string `pg:",pk"`
	MachineID int64
	Address   string
	Port      int64
	Key       string
}

const AccessPointControl = "control"
const AccessPointStatistics = "statistics"

// GetAllAccessPointsByAppID returns all access points for an app with given ID.
func GetAllAccessPointsByAppID(db *dbops.PgDB, appID int64) ([]*AccessPoint, error) {
	var accessPoints []*AccessPoint

	err := db.Model(&accessPoints).
		Where("app_id = ?", appID).
		OrderExpr("access_point.type ASC").
		Select()

	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with getting access points for app id %d", appID)
		return accessPoints, err
	}
	return accessPoints, nil
}

// AppendAccessPoint is an utility function that appends an access point to a
// list.
func AppendAccessPoint(list []*AccessPoint, tp, address, key string, port int64) []*AccessPoint {
	list = append(list, &AccessPoint{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	})
	return list
}
