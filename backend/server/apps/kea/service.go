package kea

import (
	errors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

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
func DetectHAServices(dbi dbops.DBI, daemon *dbmodel.Daemon) ([]dbmodel.Service, error) {
	// If it is not a Kea daemon  or it lacks the configuration. Nothing to do.
	if daemon.KeaDaemon == nil || daemon.KeaDaemon.KeaDHCPDaemon == nil || daemon.KeaDaemon.Config == nil {
		return []dbmodel.Service{}, nil
	}
	// No HA hook library. Nothing to do for this daemon.
	_, params, ok := daemon.KeaDaemon.Config.GetHookLibraries().GetHAHookLibrary()
	if !ok {
		return []dbmodel.Service{}, nil
	}
	// Must have HA relationships.
	relationships := params.GetAllRelationships()
	if len(relationships) == 0 {
		return []dbmodel.Service{}, nil
	}
	var (
		dbServices []dbmodel.Service
		services   []dbmodel.Service
	)
	for _, relationship := range relationships {
		// Make sure that all required parameters are set.
		if !relationship.IsValid() {
			return []dbmodel.Service{}, errors.New("invalid HA relationship configuration found")
		}
		// Find which peer matches this server name.
		var thisPeer *keaconfig.Peer
		for i, peer := range relationship.Peers {
			if *peer.Name == *relationship.ThisServerName {
				thisPeer = &relationship.Peers[i]
				break
			}
		}
		// If we can't find our peer. There is nothing we can do.
		if thisPeer == nil {
			return []dbmodel.Service{}, errors.Errorf("HA relationship configuration lacks the %s server's peer settings", *relationship.ThisServerName)
		}
		// Lazily get the existing services.
		if dbServices == nil {
			var err error
			dbServices, err = dbmodel.GetDetailedAllServices(dbi)
			if err != nil {
				return []dbmodel.Service{}, errors.WithMessage(err, "failed to get the existing services while detecting new services")
			}
		}
		// Try to match the service with our relationship.
		var service *dbmodel.Service
	SERVICE_MATCH_LOOP:
		for i := range dbServices {
			if len(dbServices[i].Daemons) == 0 {
				continue
			}
			if dbServices[i].HAService != nil && dbServices[i].HAService.HAType == daemon.Name {
				for _, peer := range relationship.Peers {
					if *peer.Name == dbServices[i].HAService.Relationship {
						service = &dbServices[i]
						break SERVICE_MATCH_LOOP
					}
				}
			}
		}
		// If we haven't found the matching service, let's create one.
		if service == nil {
			service = &dbmodel.Service{
				BaseService: dbmodel.BaseService{
					Daemons: []*dbmodel.Daemon{daemon},
				},
				HAService: &dbmodel.BaseHAService{
					HAType:       daemon.Name,
					Relationship: *thisPeer.Name,
				},
			}
		}
		if len(service.HAService.HAMode) == 0 {
			service.HAService.HAMode = *relationship.Mode
		}
		switch *(thisPeer.Role) {
		case "primary":
			service.HAService.PrimaryID = daemon.ID
		case "secondary", "standby":
			service.HAService.SecondaryID = daemon.ID
		default:
			service.HAService.BackupID = append(service.HAService.BackupID, daemon.ID)
		}
		services = append(services, *service)
	}
	return services, nil
}
