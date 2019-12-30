package kea

import (
	"fmt"
	"time"
	"strings"
	"context"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/database/model"
	"isc.org/stork/server/agentcomm"
)


func GetConfig(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (agentcomm.KeaResponseList, error) {
	caURL := fmt.Sprintf("http://localhost:%d/", dbApp.CtrlPort)

	daemons := make(agentcomm.KeaDaemons)

	for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
		if d.Active && d.Name != "ca" {
			daemons[d.Name] = true
		}
	}

	cmd, _ := agentcomm.NewKeaCommand("config-get", &daemons, &map[string]interface{}{})

	ctx2, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	responseList, err := agents.ForwardToKeaOverHttp(ctx2, caURL, cmd, dbApp.Machine.Address, dbApp.Machine.AgentPort)
	if err != nil {
		return nil, err
	}
	return responseList, nil
}


func GetDaemonHooks(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) (map[string][]string, error) {
	hooksByDaemon := make(map[string][]string)

	rspList, err := GetConfig(ctx, agents, dbApp)
	if err != nil {
		return nil, err
	}

	for _, rsp := range rspList {
		if rsp.Result != 0 {
			log.Warnf("response from daemon %s not ok: code:%d, text: %s",
				rsp.Daemon, rsp.Result, rsp.Text)
			continue
		}
		rootNodeName := strings.Title(rsp.Daemon)
		dhcp4Node, ok := (*rsp.Arguments)[rootNodeName].(map[string]interface{})
		if !ok {
			log.Warnf("missing root node %s", rootNodeName)
			continue
		}
		hookNodes, ok := dhcp4Node["hooks-libraries"].([]interface{})
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
