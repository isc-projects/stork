package kea

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// === status-get response structs ================================================

// Represents the status of the local server (the one that
// responded to the command).
type HALocalStatus struct {
	ServerName string `json:"server-name"`
	Role       string
	Scopes     []string
	State      string
}

// Represents the status of the remote server.
type HARemoteStatus struct {
	ServerName         string `json:"server-name"`
	Age                int64
	InTouch            bool `json:"in-touch"`
	Role               string
	LastScopes         []string `json:"last-scopes"`
	LastState          string   `json:"last-state"`
	CommInterrupted    *bool    `json:"communication-interrupted"`
	ConnectingClients  int64    `json:"connecting-clients"`
	UnackedClients     int64    `json:"unacked-clients"`
	UnackedClientsLeft int64    `json:"unacked-clients-left"`
	AnalyzedPackets    int64    `json:"analyzed-packets"`
}

// Represents the status of the HA enabled Kea servers.
type HAServersStatus struct {
	Local  HALocalStatus
	Remote HARemoteStatus
}

// Represent a status of a single HA relationship encapsulated in the
// high-availability list of a status-get response.
type HARelationshipStatus struct {
	HAMode    string          `json:"ha-mode"`
	HAServers HAServersStatus `json:"ha-servers"`
}

// Represents the arguments of the response to the status-get command.
// In Kea 1.7.8 the data structure of the response was modified. In the
// earlier versions of Kea, the ha-servers argument contained the status
// of the HA pair. In the later versions there is a list of the HA
// relationships under high-availability list. We support both for
// backward compatibility.
type StatusGetRespArgs struct {
	Pid    int64
	Uptime int64
	Reload int64
	// HAServers contains the HA status sent by Kea versions earlier
	// than 1.7.8.
	HAServers *HAServersStatus `json:"ha-servers"`
	// HA contains the HA status sent by Kea versions 1.7.8+.
	HA []HARelationshipStatus `json:"high-availability"`
}

// Represents a response from the single Kea server to the status-get
// command.
type StatusGetResponse struct {
	keactrl.ResponseHeader
	Arguments *StatusGetRespArgs `json:"arguments,omitempty"`
}

// Holds the status of the Kea daemon.
type daemonStatus struct {
	StatusGetRespArgs
	Daemon string
}

// Holds the status of the Kea app.
type appStatus []daemonStatus

// Instance of the puller which periodically checks the status of the Kea apps.
// Besides basic status information the High Availability status is fetched.
type HAStatusPuller struct {
	*agentcomm.PeriodicPuller
}

// Create an instance of the puller which periodically checks the status of
// the Kea apps.
func NewHAStatusPuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents) (*HAStatusPuller, error) {
	puller := &HAStatusPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Status puller",
		"kea_status_puller_interval", puller.pullData)
	if err != nil {
		return nil, err
	}
	puller.PeriodicPuller = periodicPuller
	return puller, nil
}

// Stops the timer triggering status checks.
func (puller *HAStatusPuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// This function updates HA service status based on the response from the Kea
// servers.
func updateHAServiceStatus(status *HAServersStatus, daemon *dbmodel.Daemon, service *dbmodel.BaseHAService) {
	// The status of the remote server should contain "age" value which indicates
	// how many seconds ago the status of the remote server was gathered. If this
	// value is present, calculate its timestamp.
	agePresent := false
	age, err := time.ParseDuration(fmt.Sprintf("%ds", status.Remote.Age))
	if err == nil {
		agePresent = true
	}
	// Get the current time.
	now := storkutil.UTCNow()

	var (
		primaryLastState   dbmodel.HAState
		secondaryLastState dbmodel.HAState
	)
	// Depending if this server is primary or secondary/standby, we will fill in
	// different columns of the ha_service table.
	switch daemon.ID {
	case service.PrimaryID:
		// Primary responded giving its state as "local" server's state.
		primaryLastState = status.Local.State
		service.PrimaryStatusCollectedAt = now
		service.PrimaryLastScopes = status.Local.Scopes
		service.PrimaryReachable = true
		// The state of the secondary should have been returned as "remote"
		// server's state.
		secondaryLastState = status.Remote.LastState
		if secondaryLastState != dbmodel.HAStateUnavailable {
			service.SecondaryStatusCollectedAt = now
			if agePresent {
				// If there is age, we have to shift the timestamp backwards by age.
				service.SecondaryStatusCollectedAt = service.SecondaryStatusCollectedAt.Add(-age)
			}
			service.SecondaryLastScopes = status.Remote.LastScopes
			service.SecondaryReachable = true
		} else {
			service.SecondaryLastScopes = []string{}
			service.SecondaryReachable = false
		}
		service.SecondaryCommInterrupted = status.Remote.CommInterrupted

		// Failover procedure should only be monitored during normal operation.
		switch primaryLastState {
		case dbmodel.HAStateLoadBalancing, dbmodel.HAStateHotStandby:
			service.SecondaryConnectingClients = status.Remote.ConnectingClients
			service.SecondaryUnackedClients = status.Remote.UnackedClients
			service.SecondaryUnackedClientsLeft = status.Remote.UnackedClientsLeft
			service.SecondaryAnalyzedPackets = status.Remote.AnalyzedPackets
		default:
			service.SecondaryConnectingClients = 0
			service.SecondaryUnackedClients = 0
			service.SecondaryUnackedClientsLeft = 0
			service.SecondaryAnalyzedPackets = 0
		}
	case service.SecondaryID:
		// Record the secondary/standby server's state.
		secondaryLastState = status.Local.State
		service.SecondaryStatusCollectedAt = now
		service.SecondaryLastScopes = status.Local.Scopes
		service.SecondaryReachable = true
		// The server which responded to the command was the secondary. Therefore,
		// the primary's state is given as "remote" server's state.
		primaryLastState = status.Remote.LastState

		if primaryLastState != dbmodel.HAStateUnavailable {
			service.PrimaryStatusCollectedAt = now
			if agePresent {
				// If there is age, we have to shift the timestamp backwards by age.
				service.PrimaryStatusCollectedAt = service.PrimaryStatusCollectedAt.Add(-age)
			}
			service.PrimaryLastScopes = status.Remote.LastScopes
			service.PrimaryReachable = true
		} else {
			service.PrimaryLastScopes = []string{}
			service.PrimaryReachable = false
		}
		service.PrimaryCommInterrupted = status.Remote.CommInterrupted

		// Failover procedure should only be monitored during normal operation.
		switch secondaryLastState {
		case dbmodel.HAStateLoadBalancing, dbmodel.HAStateHotStandby:
			service.PrimaryConnectingClients = status.Remote.ConnectingClients
			service.PrimaryUnackedClients = status.Remote.UnackedClients
			service.PrimaryUnackedClientsLeft = status.Remote.UnackedClientsLeft
			service.PrimaryAnalyzedPackets = status.Remote.AnalyzedPackets
		default:
			service.PrimaryConnectingClients = 0
			service.PrimaryUnackedClients = 0
			service.PrimaryUnackedClientsLeft = 0
			service.PrimaryAnalyzedPackets = 0
		}
	}
	// Finally, if any of the server's is in the partner-down state we should
	// record it as failover event.
	if primaryLastState != service.PrimaryLastState {
		if primaryLastState == dbmodel.HAStatePartnerDown {
			service.PrimaryLastFailoverAt = service.PrimaryStatusCollectedAt
		}
		service.PrimaryLastState = primaryLastState
	}
	if secondaryLastState != service.SecondaryLastState {
		if secondaryLastState == dbmodel.HAStatePartnerDown {
			service.SecondaryLastFailoverAt = service.SecondaryStatusCollectedAt
		}
		service.SecondaryLastState = secondaryLastState
	}
}

// Iterates over the slice of HA services and updates them in the database.
func (puller *HAStatusPuller) commitHAServicesStatus(appID int64, services []dbmodel.Service) {
	for i := range services {
		// Update the information about the HA service in the database.
		err := dbmodel.UpdateBaseHAService(puller.DB, services[i].HAService)
		if err != nil {
			log.Errorf("Error occurred while updating HA services status for Kea app %d: %+v", appID, err)

			continue
		}
	}
}

// Gets the status of the Kea apps and stores useful information in the database.
// The High Availability status is stored in the database for those apps which
// have the HA enabled.
func (puller *HAStatusPuller) pullData() error {
	// Get the list of all Kea apps from the database.
	apps, err := dbmodel.GetAppsByType(puller.DB, dbmodel.AppTypeKea)
	if err != nil {
		return err
	}

	var lastErr error
	appsOkCnt := 0
	appsCnt := 0
	for i := range apps {
		pulled, ok := puller.pullDataForApp(&apps[i])
		if pulled {
			appsCnt++
		}
		if ok {
			appsOkCnt++
		}
	}
	log.Printf("Completed pulling DHCP status from Kea apps: %d/%d succeeded", appsOkCnt, appsCnt)

	return lastErr
}

// Gets the status of a Kea app and stores useful information in the database.
// The High Availability status is stored in the database for those apps which
// have the HA enabled.
func (puller *HAStatusPuller) pullDataForApp(app *dbmodel.App) (bool, bool) {
	// Before contacting the DHCP server, let's check if there is any service
	// the app belongs to.
	dbServices, err := dbmodel.GetDetailedServicesByAppID(puller.DB, app.ID)
	if err != nil {
		log.Errorf("Error while getting services for Kea app %d: %+v", app.ID, err)
		return false, false
	}
	// No services for this app, so nothing to do.
	if len(dbServices) == 0 {
		return false, false
	}

	// Pick only those services for the app that have the HA type. At the
	// same time reset the values in case the server doesn't respond to the
	// command. These values will indicate that we can't say what is happening
	// with the server we failed to connect to.
	var haServices []dbmodel.Service
	for j := range dbServices {
		if dbServices[j].HAService == nil {
			continue
		}
		for _, d := range app.Daemons {
			switch d.ID {
			case dbServices[j].HAService.PrimaryID:
				dbServices[j].HAService.PrimaryLastState = dbmodel.HAStateUnavailable
				dbServices[j].HAService.PrimaryLastScopes = []string{}
				dbServices[j].HAService.PrimaryReachable = false
			case dbServices[j].HAService.SecondaryID:
				dbServices[j].HAService.SecondaryLastState = dbmodel.HAStateUnavailable
				dbServices[j].HAService.SecondaryLastScopes = []string{}
				dbServices[j].HAService.SecondaryReachable = false
			}
			haServices = append(haServices, dbServices[j])
		}
	}

	ctx := context.Background()
	// Send the status-get command to both DHCPv4 and DHCPv6 servers.
	appStatus, err := getDHCPStatus(ctx, puller.Agents, app)
	if err != nil {
		log.Errorf("Error occurred while getting Kea app %d status: %+v", app.ID, err)

		return true, false
	}
	// Go over the returned status values and match with the daemons.
	for _, status := range appStatus {
		// If no HA status, there is nothing to do.
		if status.HAServers == nil && len(status.HA) == 0 {
			continue
		}
		// Find the matching service for the returned status.
		for i := range haServices {
			if haServices[i].HAService.HAType == status.Daemon {
				service := haServices[i].HAService
				for _, daemon := range app.Daemons {
					// Update the HA service status only if the given server is primary
					// or secondary.
					if service.PrimaryID == daemon.ID || service.SecondaryID == daemon.ID {
						switch {
						case len(status.HA) == 1:
							updateHAServiceStatus(&status.HA[0].HAServers, daemon, service)
						case len(status.HA) > 1:
							for j, ha := range status.HA {
								if ha.HAServers.Local.ServerName == service.Relationship ||
									ha.HAServers.Remote.ServerName == service.Relationship {
									updateHAServiceStatus(&status.HA[j].HAServers, daemon, service)
									break
								}
							}
						case status.HAServers != nil:
							updateHAServiceStatus(status.HAServers, daemon, service)
						default:
							continue
						}
					}
				}
			}
		}
	}

	// Update the services as appropriate regardless if we successfully communicated
	// with the servers or not.
	puller.commitHAServicesStatus(app.ID, haServices)
	return true, true
}

// Sends the status-get command to Kea DHCP servers and returns this status to the caller.
func getDHCPStatus(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (appStatus, error) {
	// This command is only sent to the DHCP daemons.
	daemons := dbApp.GetActiveDHCPDaemonNames()

	// It takes no arguments, thus the last parameter is nil.
	cmd := keactrl.NewCommandBase(keactrl.StatusGet, daemons...)

	// todo: hardcoding 2s timeout is a temporary solution. We need better
	// control over the timeouts.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// The Kea response will be stored in this slice of structures.
	response := []StatusGetResponse{}

	// Send the command and receive the response.
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []keactrl.SerializableCommand{cmd}, &response)
	if err != nil {
		return nil, err
	}
	if err = cmdsResult.GetFirstError(); err != nil {
		return nil, err
	}

	// Extract the status value.
	appStatus := appStatus{}
	for _, r := range response {
		if err := r.GetError(); err != nil {
			if len(r.Daemon) > 0 {
				log.WithError(err).Warnf("status-get command failed for Kea daemon %s", r.Daemon)
			} else {
				log.WithError(err).Warn("status-get command failed")
			}
		} else if r.Arguments != nil {
			daemonStatus := daemonStatus{
				StatusGetRespArgs: *r.Arguments,
				Daemon:            r.Daemon,
			}
			appStatus = append(appStatus, daemonStatus)
		}
	}

	return appStatus, nil
}
