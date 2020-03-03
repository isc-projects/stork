package kea

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
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

// Sends the status-get command to Kea DHCP servers and returns this status to the caller.
func GetDHCPStatus(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (AppStatus, error) {
	// This command is only sent to the DHCP deamons.
	daemons, _ := agentcomm.NewKeaDaemons(dbApp.GetActiveDHCPDeamonNames()...)

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
