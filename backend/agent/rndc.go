package agent

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// CommandExecutor takes an array of strings, with the first element of the
// array being the program to call, followed by its arguments.  It returns
// the command output, and possibly an error (for example if running the
// command failed).
type CommandExecutor func([]string) ([]byte, error)

type RndcClient struct {
	execute CommandExecutor
}

const RndcKeyFile = "/etc/bind/rndc.key"

// Create an rndc client to communicate with BIND 9 named daemon.
func NewRndcClient(ce CommandExecutor) *RndcClient {
	rndcClient := &RndcClient{
		execute: ce,
	}
	return rndcClient
}

func (c *RndcClient) Call(app App, command []string) (output []byte, err error) {
	ctrl, err := getAccessPoint(app, AccessPointControl)
	if err != nil {
		return nil, err
	}

	rndcCommand := []string{"rndc", "-s", ctrl.Address, "-p", fmt.Sprintf("%d", ctrl.Port)}
	if len(ctrl.Key) > 0 {
		rndcCommand = append(rndcCommand, "-y")
		rndcCommand = append(rndcCommand, ctrl.Key)
	} else if _, err := os.Stat(RndcKeyFile); err == nil {
		rndcCommand = append(rndcCommand, "-k")
		rndcCommand = append(rndcCommand, RndcKeyFile)
	}
	rndcCommand = append(rndcCommand, command...)
	log.Debugf("rndc: %+v", rndcCommand)

	return c.execute(rndcCommand)
}
