package kea

import (
	dbmodel "isc.org/stork/server/database/model"
)

func DetectHAServices(dbApp *dbmodel.App) (services []dbmodel.Service) {
	appKea, ok := dbApp.Details.(dbmodel.AppKea)
	if !ok {
		return services
	}

	for _, d := range appKea.Daemons {
		if d.Config == nil || (d.Name != "dhcp4" && d.Name != "dhcp6") {
			continue
		}

		if _, _, ok := d.Config.GetHAHooksLibrary(); ok {
			// todo: check if there is already a matching service in the db.

			service := dbmodel.Service{
				HAService: &dbmodel.BaseHAService{
					HAType: d.Name,
				},
			}
			services = append(services, service)
		}
	}

	return services
}
