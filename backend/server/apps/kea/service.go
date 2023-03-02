package kea

import (
	keaconfig "isc.org/stork/appcfg/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Checks if the specified Kea daemon belongs to a given HA service.
// This is done by matching the HA configuration of the given daemon
// with the HA configurations of the other daemons already associated
// with the service. In particular, the HA mode must match and for
// the peers' configurations the server names, URLs and roles must
// match.
func daemonBelongsToHAService(daemon *dbmodel.Daemon, service *dbmodel.Service) bool {
	// If there are no daemons associated with the service, there is
	// nothing we can compare the daemon's configuration with.
	if len(service.Daemons) == 0 {
		return false
	}

	// Get the daemon's HA configuration and check if it is set.
	_, daemonConfigHA, ok := daemon.KeaDaemon.Config.GetHookLibraries().GetHAHookLibrary()
	if !ok || !daemonConfigHA.GetFirst().IsValid() {
		return false
	}

	// We have to iterate over the daemons already associated with the service and
	// compare their configurations with the configuration of the daemon.
	for _, sd := range service.Daemons {
		// Do not compare the daemon to itself.
		if sd.ID == daemon.ID {
			continue
		}

		// Rule out all daemons belonging to the service which aren't of
		// the Kea DHCP type. Also those which names do not match.
		if sd.KeaDaemon == nil || sd.KeaDaemon.KeaDHCPDaemon == nil || sd.Name != daemon.Name {
			continue
		}

		// Get the HA configuration of the daemon belonging to the service.
		var serviceDaemonConfigHA keaconfig.HALibraryParams
		_, serviceDaemonConfigHA, ok = sd.KeaDaemon.Config.GetHookLibraries().GetHAHookLibrary()
		if !ok || !serviceDaemonConfigHA.GetFirst().IsValid() || (*daemonConfigHA.GetFirst().Mode != *serviceDaemonConfigHA.GetFirst().Mode) {
			// There is something wrong with the service or the mode is not matching.
			// This service is not matching.
			return false
		}

		// Now we have to compare the peers' configurations.
		for _, servicePeer := range serviceDaemonConfigHA.GetFirst().Peers {
			// For the given peer in the service let's find the corresponding one
			// specified in the daemons's configuration.
			ok = false
			for _, daemonPeer := range daemonConfigHA.GetFirst().Peers {
				if (*daemonPeer.Name == *servicePeer.Name) &&
					(*daemonPeer.URL == *servicePeer.URL) &&
					(*daemonPeer.Role == *servicePeer.Role) {
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

	// Passed all checks that could possibly eliminate the daemon from the service.
	return true
}

// Parses High Availability configuration of the given Kea daemon and matches that
// configuration with existing services. If no matching service is found, it
// is created and returned. This function neither creates nor updates any
// services in the database. It is up to the caller of this function to
// perform such updates based on the returned services by the function.
// It is possible to check whether the returned service is a new instance
// or an instance already present in the database by calling the
// Service.IsNew() function. If this is a new service, the caller should
// call dbmodel.AddService() to add the new service and the associations of
// the daemons with this service. Otherwise, the caller should first call the
// UpdateBaseHAService function using the service returned by this function.
// Next, AddDaemonToService() should be called to associate the daemon with the
// service in the database. A single daemon may belong to multiple services.
func DetectHAServices(dbi dbops.DBI, daemon *dbmodel.Daemon) (services []dbmodel.Service) {
	// We only detect HA services for DHCP daemons. Other daemons do not support it.
	if daemon.KeaDaemon == nil || daemon.KeaDaemon.KeaDHCPDaemon == nil || daemon.KeaDaemon.Config == nil {
		return services
	}

	// Check if the configuration contains any HA configuration.
	if _, params, ok := daemon.KeaDaemon.Config.GetHookLibraries().GetHAHookLibrary(); ok {
		// Make sure that the required parameters are set.
		if !params.GetFirst().IsValid() {
			return services
		}

		// HA configuration must contain this-server-name parameter which indicates
		// which of the peers' configurations belongs to it. Let's iterate over
		// the configured peers to identify the one.
		index := -1
		for i, p := range params.GetFirst().Peers {
			// this-server-name matches one of the peers. Remember which one.
			if *p.Name == *params.GetFirst().ThisServerName {
				index = i
			}
		}

		// This server not found.
		if index < 0 {
			return services
		}

		// This server configuration found.
		thisServer := params.GetFirst().Peers[index]

		// Next, check if there are any existing services matching this daemon.
		index = -1
		dbServices, _ := dbmodel.GetDetailedAllServices(dbi)
		for i, service := range dbServices {
			if (service.HAService != nil) &&
				(service.HAService.HAType == daemon.Name) &&
				daemonBelongsToHAService(daemon, &dbServices[i]) {
				index = i
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
					HAType: daemon.Name,
				},
			}

			service.Daemons = append(service.Daemons, daemon)
		}

		// Set HA mode, if not set yet.
		if len(service.HAService.HAMode) == 0 {
			service.HAService.HAMode = *params.GetFirst().Mode
		}

		// Depending on the role of this server we will be setting different column
		// of the HA service column.
		switch *(thisServer.Role) {
		case "primary":
			service.HAService.PrimaryID = daemon.ID
		case "secondary", "standby":
			service.HAService.SecondaryID = daemon.ID
		default:
			service.HAService.BackupID = append(service.HAService.BackupID, daemon.ID)
		}

		services = append(services, service)
	}

	return services
}
