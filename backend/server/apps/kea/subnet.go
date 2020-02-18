package kea

import (
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

func DetectSubnets(db *dbops.PgDB, app *dbmodel.App) (subnets []dbmodel.Subnet) {
	// If this is not Kea application there is nothing to do.
	appKea, ok := app.Details.(dbmodel.AppKea)
	if !ok {
		return subnets
	}

	// If this is the Kea application, we need to iterate over the DHCP daemons and
	// match their configuration against the services.
	for _, d := range appKea.Daemons {
		if d.Config == nil {
			continue
		}

		var subnetParamName string
		switch d.Name {
		case "dhcp4":
			subnetParamName = "subnet4"
		case "dhcp6":
			subnetParamName = "subnet6"
		default:
			continue
		}

		if subnetList, ok := d.Config.GetTopLevelList(subnetParamName); ok {
			for _, s := range subnetList {
				if subnetMap, ok := s.(map[string]interface{}); ok {
					subnet := dbmodel.NewSubnet(&subnetMap)
					subnets = append(subnets, *subnet)
				}
			}
		}
	}
	return subnets
}
