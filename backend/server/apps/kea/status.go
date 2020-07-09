package kea

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

const (
	HAStatusUnavailable   = "unavailable"
	HAStatusLoadBalancing = "load-balancing"
	HAStatusHotStandby    = "hot-standby"
)

// === status-get response structs ================================================

// Represents the status of the local server (the one that
// responded to the command).
type HALocalStatus struct {
	Role   string
	Scopes []string
	State  string
}

// Represents the status of the remote server.
type HARemoteStatus struct {
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
	agentcomm.KeaResponseHeader
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
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Status",
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
		primaryLastState   string
		secondaryLastState string
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
		if secondaryLastState != HAStatusUnavailable {
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
		case HAStatusLoadBalancing, HAStatusHotStandby:
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
		// The server which responded to the command was the secondary. Therfore,
		// the primary's state is given as "remote" server's state.
		primaryLastState = status.Remote.LastState

		if primaryLastState != HAStatusUnavailable {
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
		case HAStatusLoadBalancing, HAStatusHotStandby:
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
		if primaryLastState == "partner-down" {
			service.PrimaryLastFailoverAt = service.PrimaryStatusCollectedAt
		}
		service.PrimaryLastState = primaryLastState
	}
	if secondaryLastState != service.SecondaryLastState {
		if secondaryLastState == "partner-down" {
			service.SecondaryLastFailoverAt = service.SecondaryStatusCollectedAt
		}
		service.SecondaryLastState = secondaryLastState
	}
}

// Gets the status of the Kea apps and stores useful information in the database.
// The High Availability status is stored in the database for those apps which
// have the HA enabled.
func (puller *HAStatusPuller) pullData() (int, error) {
	// Get the list of all Kea apps from the database.
	apps, err := dbmodel.GetAppsByType(puller.Db, dbmodel.AppTypeKea)
	if err != nil {
		return 0, err
	}

	var lastErr error
	appsOkCnt := 0
	appsCnt := 0
	for i := range apps {
		// Before contacting the DHCP server, let's check if there is any service
		// the app belongs to.
		dbServices, err := dbmodel.GetDetailedServicesByAppID(puller.Db, apps[i].ID)
		if err != nil {
			log.Errorf("error while getting services for Kea app %d: %+v", apps[i].ID, err)
			continue
		}
		// No services for this app, so nothing to do.
		if len(dbServices) == 0 {
			continue
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
			for _, d := range apps[i].Daemons {
				switch d.ID {
				case dbServices[j].HAService.PrimaryID:
					dbServices[j].HAService.PrimaryLastState = HAStatusUnavailable
					dbServices[j].HAService.PrimaryLastScopes = []string{}
					dbServices[j].HAService.PrimaryReachable = false
				case dbServices[j].HAService.SecondaryID:
					dbServices[j].HAService.SecondaryLastState = HAStatusUnavailable
					dbServices[j].HAService.SecondaryLastScopes = []string{}
					dbServices[j].HAService.SecondaryReachable = false
				}
				haServices = append(haServices, dbServices[j])
			}
		}

		appsCnt++
		ctx := context.Background()
		// Send the status-get command to both DHCPv4 and DHCPv6 servers.
		appStatus, err := getDHCPStatus(ctx, puller.Agents, &apps[i])
		if err != nil {
			log.Errorf("error occurred while getting Kea app %d status: %+v", apps[i].ID, err)
		}
		// Go over the returned status values and match with the daemons.
		for _, status := range appStatus {
			// If no HA status, there is nothing to do.
			if status.HAServers == nil && len(status.HA) == 0 {
				continue
			}
			// Find the matching service for the returned status.
			index := -1

			for i := range haServices {
				if haServices[i].HAService.HAType == status.Daemon {
					index = i
				}
			}
			if index < 0 {
				continue
			}
			service := haServices[index].HAService
			for _, daemon := range apps[i].Daemons {
				// Update the HA service status only if the given server is primary
				// or secondary.
				if service.PrimaryID == daemon.ID || service.SecondaryID == daemon.ID {
					// todo: Currently Kea supports only one HA service per daemon. This
					// will change but for now it is safe to assume that only one status
					// is returned. Supporting more requires some mechanisms to
					// distinguish the HA relationships which should be first designed
					// on the Kea side.
					if len(status.HA) > 0 {
						updateHAServiceStatus(&status.HA[0].HAServers, daemon, service)
					} else if status.HAServers != nil {
						updateHAServiceStatus(status.HAServers, daemon, service)
					}
				}
			}
		}

		// Update the services as appropriate regardless if we successfully communicated
		// with the servers or not.
		for j := range haServices {
			// Update the information about the HA service in the database.
			err = dbmodel.UpdateBaseHAService(puller.Db, haServices[j].HAService)
			if err != nil {
				log.Errorf("error occurred while updating HA services status for Kea app %d: %+v", apps[i].ID, err)
				continue
			}
		}

		appsOkCnt++
	}
	log.Printf("completed pulling DHCP status from Kea apps: %d/%d succeeded", appsOkCnt, appsCnt)
	return appsOkCnt, lastErr
}

// Sends the status-get command to Kea DHCP servers and returns this status to the caller.
func getDHCPStatus(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (appStatus, error) {
	// This command is only sent to the DHCP deamons.
	daemons, _ := agentcomm.NewKeaDaemons(dbApp.GetActiveDHCPDaemonNames()...)

	// It takes no arguments, thus the last parameter is nil.
	cmd, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)

	// todo: hardcoding 2s timeout is a temporary solution. We need better
	// control over the timeouts.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// The Kea response will be stored in this slice of structures.
	response := []StatusGetResponse{}

	// Send the command and receive the response.
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*agentcomm.KeaCommand{cmd}, &response)
	if err != nil {
		return nil, err
	}
	if cmdsResult.Error != nil && len(cmdsResult.CmdsErrors) == 0 {
		return nil, cmdsResult.Error
	}
	if cmdsResult.CmdsErrors[0] != nil {
		return nil, cmdsResult.CmdsErrors[0]
	}

	// Extract the status value.
	appStatus := appStatus{}
	for _, r := range response {
		if r.Result != 0 && (len(r.Daemon) > 0) {
			log.Warnf("status-get command failed for Kea daemon %s", r.Daemon)
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
