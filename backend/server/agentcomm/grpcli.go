package agentcomm

import (
	"time"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"

	"isc.org/stork/api"
)

// Get version from agent.
func (agents *ConnectedAgents) GetVersion(address string) (string, error) {
	// Find agent in map.
	agent, err := agents.GetConnectedAgent(address)
	if err != nil {
		return "", err
	}

	// Call agent for version.
	ctx, _ := context.WithTimeout(context.Background(), 10 * time.Second)
	ver, err := agent.Client.GetVersion(ctx, &agentapi.GetVersionReq{})
	if err != nil {
		// Problem with connection, try to reconnect and retry the call
		log.Infof("problem with connection to agent %v, reconnecting", err)
		err2 := agent.MakeGrpcConnection()
		if err2 != nil {
			return "", errors.Wrap(err2, "problem with connection to agent")
		}
		ctx, _ = context.WithTimeout(context.Background(), 10 * time.Second)
		ver, err = agent.Client.GetVersion(ctx, &agentapi.GetVersionReq{})
		if err != nil {
			return "", errors.Wrap(err, "problem with connection to agent")
		}
	}

	log.Printf("version returned is %s", ver.Version)

	return ver.Version, nil
}
