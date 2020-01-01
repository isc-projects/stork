package kea

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/database/model"
)

// Retrieve configuration of the selected Kea deamons using the config-get
// command.
func GetConfig(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, daemons *agentcomm.KeaDaemons) (agentcomm.KeaResponseList, error) {
	// We assume that the Kea Control Agent runs on the same machine as
	// the Stork Agent. Thus, we use localhost to communicate with the CA.
	caURL := fmt.Sprintf("http://localhost:%d/", dbApp.CtrlPort)

	// prepare the command
	cmd, _ := agentcomm.NewKeaCommand("config-get", daemons, nil)

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// send the command to daemons through agent and return response list
	responseList := agentcomm.KeaResponseList{}
	err := agents.ForwardToKeaOverHttp(ctx2, caURL, dbApp.Machine.Address, dbApp.Machine.AgentPort, cmd, &responseList)
	if err != nil {
		return nil, err
	}
	return responseList, nil
}

// Get list of hooks for all DHCP daemons of the given Kea application.
// It uses GetConfig function.
func GetDaemonHooks(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (map[string][]string, error) {
	hooksByDaemon := make(map[string][]string)

	// find out which daemons are active
	daemons := make(agentcomm.KeaDaemons)
	for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
		if d.Active && (d.Name == "dhcp4" || d.Name == "dhcp6") {
			daemons[d.Name] = true
		}
	}

	// get configs from daemons
	rspList, err := GetConfig(ctx, agents, dbApp, &daemons)
	if err != nil {
		return nil, err
	}

	// go through response list with configs from each daemon and retrieve their hooks lists
	for _, rsp := range rspList {
		if rsp.Result != 0 {
			log.Warnf("getting installed hooks from daemon %s failed with error code: %d, text: %s",
				rsp.Daemon, rsp.Result, rsp.Text)
			continue
		}
		rootNodeName := strings.Title(rsp.Daemon)
		dhcpNode, ok := (*rsp.Arguments)[rootNodeName].(map[string]interface{})
		if !ok {
			log.Warnf("missing root node %s", rootNodeName)
			continue
		}
		hookNodes, ok := dhcpNode["hooks-libraries"].([]interface{})
		if !ok {
			continue
		}
		hooks := []string{}
		for _, hookNode := range hookNodes {
			hookNode2, ok := hookNode.(map[string]interface{})
			if !ok {
				continue
			}
			library, ok := hookNode2["library"].(string)
			if ok {
				hooks = append(hooks, library)
			}
		}
		hooksByDaemon[rsp.Daemon] = hooks
	}

	return hooksByDaemon, nil
}
