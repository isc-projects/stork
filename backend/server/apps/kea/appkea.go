package kea

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Get list of hooks for all DHCP daemons of the given Kea application.
func GetDaemonHooks(dbApp *dbmodel.App) map[string][]string {
	hooksByDaemon := make(map[string][]string)

	// go through response list with configs from each daemon and retrieve their hooks lists
	for _, dmn := range dbApp.Details.(dbmodel.AppKea).Daemons {
		if dmn.Config == nil {
			continue
		}

		libraries := dmn.Config.GetHooksLibraries()
		hooks := []string{}
		for _, library := range libraries {
			hooks = append(hooks, library.Library)
		}
		hooksByDaemon[dmn.Name] = hooks
	}

	return hooksByDaemon
}

// === CA config-get response structs ================================================

type SocketData struct {
	SocketName string `json:"socket-name"`
	SocketType string `json:"socket-type"`
}

type ControlSocketsData struct {
	D2      *SocketData
	Dhcp4   *SocketData
	Dhcp6   *SocketData
	NetConf *SocketData
}

type ControlAgentData struct {
	ControlSockets *ControlSocketsData `json:"control-sockets"`
}

type CAConfigGetRespArgs struct {
	ControlAgent *ControlAgentData `json:"Control-agent"`
}

type CAConfigGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *CAConfigGetRespArgs
}

// === version-get response structs ===============================================

type VersionGetRespArgs struct {
	Extended string
}

type VersionGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *VersionGetRespArgs `json:"arguments,omitempty"`
}

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
	Age        int64
	InTouch    bool `json:"in-touch"`
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
// response (i.e. the Kea server has HA disabled).
type StatusGetRespArgs struct {
	Pid       int64
	Uptime    int64
	Reload    int64
	HAServers *HAServersStatus `json:"ha-servers"`
}

type StatusGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *StatusGetRespArgs `json:"arguments,omitempty"`
}

// Get state of Kea application Control Agent using ForwardToKeaOverHTTP function.
// The state, that is stored into dbApp, includes: version and config of CA.
// It also returns:
// - list of all Kea daemons
// - list of DHCP daemons (dhcpv4 and/or dhcpv6)
func getStateFromCA(ctx context.Context, agents agentcomm.ConnectedAgents, caURL string, dbApp *dbmodel.App, daemonsMap map[string]*dbmodel.KeaDaemon) (agentcomm.KeaDaemons, agentcomm.KeaDaemons, error) {
	// prepare the command to get config and version from CA
	cmds := []*agentcomm.KeaCommand{
		{
			Command: "version-get",
		},
		{
			Command: "config-get",
		},
	}

	// get version and config from CA
	versionGetResp := []VersionGetResponse{}
	caConfigGetResp := []CAConfigGetResponse{}

	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, caURL, cmds, &versionGetResp, &caConfigGetResp)
	if err != nil {
		return nil, nil, err
	}
	if cmdsResult.Error != nil {
		return nil, nil, cmdsResult.Error
	}

	// process the response from CA
	daemonsMap["ca"] = &dbmodel.KeaDaemon{
		Name:   "ca",
		Active: true,
	}

	if cmdsResult.CmdsErrors[0] == nil {
		vRsp := versionGetResp[0]
		dmn := daemonsMap["ca"]
		if vRsp.Result != 0 {
			dmn.Active = false
			log.Warnf("problem with version-get from CA: %s", vRsp.Text)
		} else {
			dmn.Version = vRsp.Text
			dbApp.Meta.Version = vRsp.Text
			if vRsp.Arguments != nil {
				dmn.ExtendedVersion = vRsp.Arguments.Extended
			}
		}
	} else {
		log.Warnf("problem with version-get response from CA: %s", cmdsResult.CmdsErrors[0])
	}

	allDaemons := make(agentcomm.KeaDaemons)
	dhcpDaemons := make(agentcomm.KeaDaemons)
	if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets != nil {
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.Dhcp4 != nil {
			allDaemons["dhcp4"] = true
			dhcpDaemons["dhcp4"] = true
		}
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.Dhcp6 != nil {
			allDaemons["dhcp6"] = true
			dhcpDaemons["dhcp6"] = true
		}
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.D2 != nil {
			allDaemons["d2"] = true
		}
	}

	return allDaemons, dhcpDaemons, nil
}

// Get state of Kea application daemons (beside Control Agent) using ForwardToKeaOverHTTP function.
// The state, that is stored into dbApp, includes: version, config and runtime state of indicated Kea daemons.
func getStateFromDaemons(ctx context.Context, agents agentcomm.ConnectedAgents, caURL string, dbApp *dbmodel.App, daemonsMap map[string]*dbmodel.KeaDaemon, allDaemons agentcomm.KeaDaemons, dhcpDaemons agentcomm.KeaDaemons) error {
	now := storkutil.UTCNow()

	// issue 3 commands to Kea daemons at once to get their state
	cmds := []*agentcomm.KeaCommand{
		{
			Command: "version-get",
			Daemons: &allDaemons,
		},
		{
			Command: "status-get",
			Daemons: &dhcpDaemons,
		},
		{
			Command: "config-get",
			Daemons: &allDaemons,
		},
	}

	versionGetResp := []VersionGetResponse{}
	statusGetResp := []StatusGetResponse{}
	configGetResp := []agentcomm.KeaResponse{}

	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, caURL, cmds, &versionGetResp, &statusGetResp, &configGetResp)
	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	for name := range allDaemons {
		daemonsMap[name] = &dbmodel.KeaDaemon{
			Name:   name,
			Active: true,
		}
	}

	// process version-get responses
	err = cmdsResult.CmdsErrors[0]
	if err != nil {
		log.Warnf("problem with version-get response: %s", err)
	} else {
		for _, vRsp := range versionGetResp {
			dmn := daemonsMap[vRsp.Daemon]
			if vRsp.Result != 0 {
				dmn.Active = false
				log.Warnf("problem with version-get and kea daemon %s: %s", vRsp.Daemon, vRsp.Text)
				continue
			}

			dmn.Version = vRsp.Text
			if vRsp.Arguments != nil {
				dmn.ExtendedVersion = vRsp.Arguments.Extended
			}
		}
	}

	// process status-get responses
	err = cmdsResult.CmdsErrors[1]
	if err != nil {
		log.Warnf("problem with status-get response: %s", err)
	} else {
		for _, sRsp := range statusGetResp {
			dmn := daemonsMap[sRsp.Daemon]
			if sRsp.Result != 0 {
				dmn.Active = false
				log.Warnf("problem with status-get and kea daemon %s: %s", sRsp.Daemon, sRsp.Text)
				continue
			}

			if sRsp.Arguments != nil {
				dmn.Uptime = sRsp.Arguments.Uptime
				dmn.ReloadedAt = now.Add(time.Second * time.Duration(-sRsp.Arguments.Reload))
				// TODO: HA status
			}
		}
	}

	// process config-get responses
	err = cmdsResult.CmdsErrors[2]
	if err != nil {
		log.Warnf("problem with config-get response: %s", err)
	} else {
		for _, cRsp := range configGetResp {
			dmn := daemonsMap[cRsp.Daemon]
			if cRsp.Result != 0 {
				dmn.Active = false
				log.Warnf("problem with config-get and kea daemon %s: %s", cRsp.Daemon, cRsp.Text)
				continue
			}

			dmn.Config = dbmodel.NewKeaConfig(cRsp.Arguments)
		}
	}

	return nil
}

// Get state of Kea application daemons using ForwardToKeaOverHTTP function.
// The state, that is stored into dbApp, includes: version, config and runtime state of indicated Kea daemons.
func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) {
	// prepare URL to CA
	caURL := storkutil.HostWithPortURL(dbApp.CtrlAddress, dbApp.CtrlPort)

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// get state from CA
	daemonsMap := map[string]*dbmodel.KeaDaemon{}
	allDaemons, dhcpDaemons, err := getStateFromCA(ctx2, agents, caURL, dbApp, daemonsMap)

	// if not problems then now get state from the rest of Kea daemons
	if err == nil {
		err = getStateFromDaemons(ctx2, agents, caURL, dbApp, daemonsMap, allDaemons, dhcpDaemons)
		if err != nil {
			log.Warnf("problem with getting state from kea daemons: %s", err)
		}
	} else {
		log.Warnf("problem with getting state from kea CA: %s", err)
	}

	// store all collected details in app db record
	keaApp := dbmodel.AppKea{}
	dbApp.Active = true
	for name := range daemonsMap {
		dmn := daemonsMap[name]
		// if all daemons are active then whole app is active
		dbApp.Active = dbApp.Active && dmn.Active

		keaApp.Daemons = append(keaApp.Daemons, dmn)
	}
	dbApp.Details = keaApp
}
