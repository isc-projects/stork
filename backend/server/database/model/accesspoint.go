package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// A structure reflecting the access_point SQL table.
type AccessPoint struct {
	AppID             int64  `pg:",pk"`
	Type              string `pg:",pk"`
	MachineID         int64
	Address           string
	Port              int64
	Key               string
	UseSecureProtocol bool `pg:",use_zero"`
}

// Valid kinds of the access points.
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

// AppendAccessPoint is an utility function that appends an access point to a
// list.
func AppendAccessPoint(list []*AccessPoint, tp, address, key string, port int64, useSecureProtocol bool) []*AccessPoint {
	list = append(list, &AccessPoint{
		Type:              tp,
		Address:           address,
		Port:              port,
		Key:               key,
		UseSecureProtocol: useSecureProtocol,
	})
	return list
}

// Get an access point by app id and access point type.
func GetAccessPointByID(db dbops.DBI, appID int64, accessPointType string) (*AccessPoint, error) {
	accessPoint := &AccessPoint{AppID: appID, Type: accessPointType}
	err := db.Model(accessPoint).WherePK().Select()

	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(
			err,
			"problem getting access point of app: %d and with type: %s",
			appID,
			accessPointType,
		)
	}
	return accessPoint, nil
}
