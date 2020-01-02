package kea

import (
	"context"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database/model"
	"isc.org/stork/util"
	"time"
)

// Represents the status of the local server (the one that
// responded to the command).
type HALocalStatus struct {
	Role   string
	Scopes []string
	State  string
}

// Represents the status of the remote server.
type HARemoteStatus struct {
	Age        int64
	InTouch    bool     `json:"in-touch"`
	Role       string
	LastScopes []string `json:"last-scopes"`
	LastState  string   `json:"last-state"`
}

// Represents the status of the HA enabled Kea servers.
type HAServersStatus struct {
	Local  HALocalStatus
	Remote HARemoteStatus
}

// Represents a response from the single Kea server to the status-get
// command. The HAServers value is nil if it is not present in the
// response.
type Status struct {
	Pid               int64
	Uptime            int64
	Reload            int64
	HAServers         *HAServersStatus `json:"ha-servers"`
	Daemon            string           `json:"-"`
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

	// The command will be forwarded from the agent to Kea CA on the same host.
	url := storkutil.LocalHostWithPort(dbApp.CtrlPort)

	// The Kea response will be stored in this slice of structures.
	response := []struct {
		agentcomm.KeaResponseHeader
		Arguments *Status
	}{}

	// Send the command and receive the response.
	err := agents.ForwardToKeaOverHttp(ctx, url, dbApp.Machine.Address, dbApp.Machine.AgentPort, cmd, &response)
	if err != nil {
		return nil, err
	}

	// Extract the status value.
	appStatus := AppStatus{}
	for _, r := range response {
		appStatus = append(appStatus, *r.Arguments)
	}

	return appStatus, nil
}

