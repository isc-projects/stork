package kea

import (
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Checks if the specified Kea app belongs to a given HA service.
// This is done by matching the HA configuration of the given app
// with the HA configurations of the other apps already associated
// with the service. In particular, the HA mode must match and for
// the peers' configurations the server names, URLs and roles must
// match.
func appBelongsToHAService(app *dbmodel.App, service *dbmodel.Service) bool {
	// If the service or app is nil, service is blank or if the app is not Kea then the app
	// surely doesn't belong to the service.
	if service.HAService == nil || len(service.Apps) == 0 || app == nil || app.Type != "kea" {
		return false
	}

	// Try to cast the App to AppKea. If that fails we bail again.
	var (
		appKea dbmodel.AppKea
		ok     bool
	)
	if appKea, ok = app.Details.(dbmodel.AppKea); !ok {
		return false
	}

	var index int = -1
	for i, d := range appKea.Daemons {
		if d.Name == service.HAService.HAType {
			index = i
		}
	}

	// Failed to find a daemon matching the service HA type. Nothing more to do.
	if index < 0 {
		return false
	}

	// Successfully found the daemon. Get its instance.
	appDaemon := appKea.Daemons[index]

	// Get the applications's HA configuration and check if it is set.
	var appConfigHA dbmodel.KeaConfigHA
	_, appConfigHA, ok = appDaemon.Config.GetHAHooksLibrary()
	if !ok || !appConfigHA.IsSet() {
		return false
	}

	// We have all required information about the app. Now we have to
	// iterate over the apps already associated with the service and
	// compare their configurations with the configuration of the
	// app.
	for _, sa := range service.Apps {
		// Do not compare the app to itself.
		if sa.ID == app.ID {
			continue
		}

		var serviceAppKea dbmodel.AppKea
		if serviceAppKea, ok = sa.Details.(dbmodel.AppKea); !ok {
			continue
		}

		// For the app belonging to the service, let's iterate over the daemons
		// to compare configurations.
		for _, serviceDaemon := range serviceAppKea.Daemons {
			// Daemon doesn't match the service type. They must be both 'dhcp4'
			// or 'dhcp6'.
			if serviceDaemon.Name != appDaemon.Name {
				continue
			}

			// Get the HA configuration of the app belonging to the service.
			var serviceDaemonConfigHA dbmodel.KeaConfigHA
			_, serviceDaemonConfigHA, ok = serviceDaemon.Config.GetHAHooksLibrary()
			if !ok || !serviceDaemonConfigHA.IsSet() || (*appConfigHA.Mode != *serviceDaemonConfigHA.Mode) {
				// There is something wrong with the service or the mode is not matching.
				// This service is not matching.
				return false
			}

			// Now we have to compare the peers' configurations.
			for _, servicePeer := range serviceDaemonConfigHA.Peers {
				// For the given peer in the service let's find the corresponding one
				// specified in the app's configuration.
				ok = false
				for _, appPeer := range appConfigHA.Peers {
					if (*appPeer.Name == *servicePeer.Name) &&
						(*appPeer.URL == *servicePeer.URL) &&
						(*appPeer.Role == *servicePeer.Role) {
						// Match found.
						ok = true
						break
					}
				}
				// Peer not found on the URL is not matching.
				if !ok {
					return false
				}
			}
		}
	}

	// Passed all checks that could possibly eliminate the app from the service.
	return true
}

// Parses High Availability configuration of the given Kea app and matches that
// configuration with existing services. If no matching service is found, it
// is created and returned. This function neither creates nor updates any
// services in the database. It is up to the caller of this function to
// perform such updates based on the returned services by the function.
// It is possible to check whether the returned service is a new instance
// or an instance already present in the database by calling the
// Service.IsNew() function. If this is a new service, the caller should
// call dbmodel.AddService() to add the new service and the associations of
// the apps with this service. Otherwise, the caller should first call the
// UpdateBaseHAService function using the service returned by this function.
// Next, AddAppToService() should be called to associate the app with the
// service in the database. A single app may belong to multiple services,
// e.g. an app can belong to Kea DHCPv4 HA service and Kea DHCPv6 HA service.
func DetectHAServices(db *dbops.PgDB, dbApp *dbmodel.App) (services []dbmodel.Service) {
	// If this is not Kea application it does not belong to HA service.
	appKea, ok := dbApp.Details.(dbmodel.AppKea)
	if !ok {
		return services
	}

	// If this is the Kea application, we need to iterate over the DHCP daemons and
	// match their configuration against the services.
	for _, d := range appKea.Daemons {
		// We only detct HA services for DHCP daemons. Other daemons do not support it.
		if d.Config == nil || (d.Name != "dhcp4" && d.Name != "dhcp6") {
			continue
		}

		// Check if the configuration contains any HA configuration.
		if _, params, ok := d.Config.GetHAHooksLibrary(); ok {
			// Make sure that the required parameters are set.
			if !params.IsSet() {
				continue
			}

			// HA configuration must contain this-server-name parameter which indicates
			// which of the peers' configurations belongs to it. Let's iterate over
			// the configured peers to identify the one.
			var index int = -1
			for i, p := range params.Peers {
				// this-server-name matches one of the peers. Remember which one.
				if *p.Name == *params.ThisServerName {
					index = i
				}
			}

			// This server not found.
			if index < 0 {
				continue
			}

			// This server configuration found.
			thisServer := params.Peers[index]

			// Next, check if there are any existing services for the URLs found
			// in this configuration.
			var dbServices []dbmodel.Service
			index = -1
			for _, p := range params.Peers {
				host, port := storkutil.ParseURL(*p.URL)
				dbServices, _ = dbmodel.GetServicesByAppCtrlAddressPort(db, host, port)
				index = -1
				for i, service := range dbServices {
					s := service
					if (service.HAService != nil) &&
						(service.HAService.HAType == d.Name) &&
						appBelongsToHAService(dbApp, &s) {
						index = i
						break
					}
				}
				if index >= 0 {
					break
				}
			}

			var service dbmodel.Service
			if index >= 0 {
				// Service found.
				service = dbServices[index]
			} else {
				// No service found in the db, so let's create one.
				service = dbmodel.Service{
					HAService: &dbmodel.BaseHAService{
						HAType: d.Name,
					},
				}

				service.Apps = append(service.Apps, dbApp)
			}

			// Set HA mode, if not set yet.
			if len(service.HAService.HAMode) == 0 {
				service.HAService.HAMode = *params.Mode
			}

			// Depending on the role of this server we will be setting different column
			// of the HA service column.
			switch *(thisServer.Role) {
			case "primary":
				service.HAService.PrimaryID = dbApp.ID
			case "secondary", "standby":
				service.HAService.SecondaryID = dbApp.ID
			default:
				service.HAService.BackupID = append(service.HAService.BackupID, dbApp.ID)
			}

			services = append(services, service)
		}
	}

	return services
}
