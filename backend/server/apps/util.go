package apps

import (
	"fmt"

	dbmodel "isc.org/stork/server/database/model"
)

// GetAccessPoint returns the access point of the given app and given access
// point type.
func GetAccessPoint(app *dbmodel.App, accessType string) (ap dbmodel.AccessPoint, err error) {
	for _, point := range app.AccessPoints {
		if point.Type == accessType {
			return point, nil
		}
	}
	return ap, fmt.Errorf("no access point of type %s found for app id %d", accessType, app.ID)
}
