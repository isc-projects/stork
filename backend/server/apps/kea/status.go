package kea

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

const (
	HAStatusUnavailable = "unavailable"
)

// Represents a response from the single Kea server to the status-get
// command. The HAServers value is nil if it is not present in the
// response.
type Status struct {
	Pid       int64
	Uptime    int64
	Reload    int64
	HAServers *HAServersStatus `json:"ha-servers"`
	Daemon    string
}

type AppStatus []Status

// Instance of the puller which periodically checks the status of the Kea apps.
// Besides basic status information the High Availability status is fetched.
type StatusPuller struct {
	*agentcomm.PeriodicPuller
}

// Create an instance of the puller which periodically checks the status of
// the Kea apps.
func NewStatusPuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents) (*StatusPuller, error) {
	haPuller := &StatusPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Status",
		"kea_status_puller_interval", haPuller.pullData)
	if err != nil {
		return nil, err
	}
	haPuller.PeriodicPuller = periodicPuller
	return haPuller, nil
}

// Stops the timer triggering status checks.
func (puller *StatusPuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// Gets the status of the Kea apps and stores useful information in the database.
// The High Availability status is stored in the database for those apps which
// have the HA enabled.
func (puller *StatusPuller) pullData() (int, error) {
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
			log.Errorf("error occurred while getting services for Kea app %d: %s", apps[i].ID, err)
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
			switch apps[i].ID {
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

		appsCnt++
		ctx := context.Background()
		// Send the status-get command to both DHCPv4 and DHCPv6 servers.
		appStatus, err := GetDHCPStatus(ctx, puller.Agents, &apps[i])
		if err != nil {
			log.Errorf("error occurred while getting Kea app %d status: %s", apps[i].ID, err)
		}
		// Go over the returned status values and match with the daemons.
		for _, status := range appStatus {
			// If no HA status, there is nothing to do.
			if status.HAServers == nil {
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
			// Check if the given app is a primary or secondary/standby server. If it is
			// not, there is nothing to do.
			service := haServices[index].HAService
			if service.PrimaryID != apps[i].ID && service.SecondaryID != apps[i].ID {
				continue
			}
			// The status of the remote server should contain "age" value which indicates
			// how many seconds ago the status of the remote server was gathered. If this
			// value is present, calculate its timestamp.
			agePresent := false
			dur, err := time.ParseDuration(fmt.Sprintf("%ds", status.HAServers.Remote.Age))
			if err == nil {
				agePresent = true
			}
			// Get the current time.
			now := time.Now().UTC()

			var (
				primaryLastState   string
				secondaryLastState string
			)
			// Depending if this server is primary or secondary/standby, we will fill in
			// different columns of the ha_service table.
			switch apps[i].ID {
			case service.PrimaryID:
				// Primary responded giving its state as "local" server's state.
				primaryLastState = status.HAServers.Local.State
				service.PrimaryStatusCollectedAt = now
				service.PrimaryLastScopes = status.HAServers.Local.Scopes
				service.PrimaryReachable = true
				// The state of the secondary should have been returned as "remote"
				// server's state.
				secondaryLastState = status.HAServers.Remote.LastState
				if secondaryLastState != HAStatusUnavailable {
					service.SecondaryStatusCollectedAt = now
					if agePresent {
						// If there is age, we have to shift the timestamp backwards by age.
						service.SecondaryStatusCollectedAt = service.SecondaryStatusCollectedAt.Add(-dur)
					}
					service.SecondaryLastScopes = status.HAServers.Remote.LastScopes
					service.SecondaryReachable = true
				} else {
					service.SecondaryLastScopes = []string{}
					service.SecondaryReachable = false
				}
			case service.SecondaryID:
				// Record the secondary/standby server's state.
				secondaryLastState = status.HAServers.Local.State
				service.SecondaryStatusCollectedAt = now
				service.SecondaryLastScopes = status.HAServers.Local.Scopes
				service.SecondaryReachable = true
				// The server which responded to the command was the secondary. Therfore,
				// the primary's state is given as "remote" server's state.
				primaryLastState = status.HAServers.Remote.LastState

				if primaryLastState != HAStatusUnavailable {
					service.PrimaryStatusCollectedAt = now
					if agePresent {
						// If there is age, we have to shift the timestamp backwards by age.
						service.PrimaryStatusCollectedAt = service.PrimaryStatusCollectedAt.Add(-dur)
					}
					service.PrimaryLastScopes = status.HAServers.Remote.LastScopes
					service.PrimaryReachable = true
				} else {
					service.PrimaryLastScopes = []string{}
					service.PrimaryReachable = false
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

		// Update the services as appropriate regardless if we successfully communicated
		// with the servers or not.
		for j := range haServices {
			// Update the information about the HA service in the database.
			err = dbmodel.UpdateBaseHAService(puller.Db, haServices[j].HAService)
			if err != nil {
				log.Errorf("error occurred while updating HA services status for Kea app %d: %s", apps[i].ID, err)
				continue
			}
		}

		appsOkCnt++
	}
	log.Printf("completed pulling DHCP status from Kea apps: %d/%d succeeded", appsOkCnt, appsCnt)
	return appsOkCnt, lastErr
}

// Sends the status-get command to Kea DHCP servers and returns this status to the caller.
func GetDHCPStatus(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (AppStatus, error) {
	// This command is only sent to the DHCP deamons.
	daemons, _ := agentcomm.NewKeaDaemons(dbApp.GetActiveDHCPDaemonNames()...)

	// It takes no arguments, thus the last parameter is nil.
	cmd, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)

	// todo: hardcoding 2s timeout is a temporary solution. We need better
	// control over the timeouts.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ctrlPoint, err := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		return nil, err
	}
	caURL := storkutil.HostWithPortURL(ctrlPoint.Address, ctrlPoint.Port)

	// The Kea response will be stored in this slice of structures.
	response := []struct {
		agentcomm.KeaResponseHeader
		Arguments *Status
	}{}

	// Send the command and receive the response.
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, caURL, []*agentcomm.KeaCommand{cmd}, &response)
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
	appStatus := AppStatus{}
	for _, r := range response {
		if r.Result != 0 && (len(r.Daemon) > 0) {
			log.Warn(errors.Errorf("status-get command failed for Kea daemon %s", r.Daemon))
		} else if r.Arguments != nil {
			appStatus = append(appStatus, *r.Arguments)
			appStatus[len(appStatus)-1].Daemon = r.Daemon
		}
	}

	return appStatus, nil
}
