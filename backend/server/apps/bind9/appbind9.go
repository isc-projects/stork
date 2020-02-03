package bind9

import (
	"context"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) {
	// Get rndc control settings
	rndcSettings := agentcomm.Bind9Control{
		CtrlAddress: dbApp.CtrlAddress,
		CtrlPort:    dbApp.CtrlPort,
		CtrlKey:     dbApp.CtrlKey,
	}

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	command := "status"
	out, err := agents.ForwardRndcCommand(ctx2, dbApp.Machine.Address, dbApp.Machine.AgentPort, rndcSettings, command)
	if err != nil {
		log.Warnf("problem with getting BIND 9 status: %s", err)
		return
	}

	bind9Daemon := dbmodel.Bind9Daemon{
		Name: "named",
	}

	// get version
	versionPtrn := regexp.MustCompile(`version:\s(.+)\n`)
	match := versionPtrn.FindStringSubmatch(out.Output)
	if match != nil {
		bind9Daemon.Version = match[1]
	} else {
		log.Warnf("cannot get BIND 9 version: unable to find version in output")
	}

	// Is the named daemon running?
	bind9Daemon.Active = false
	upPtrn := regexp.MustCompile(`server is up and running`)
	up := upPtrn.FindString(out.Output)
	if up != "" {
		bind9Daemon.Active = true
	}

	dbApp.Active = bind9Daemon.Active
	dbApp.Meta.Version = bind9Daemon.Version
	dbApp.Details = dbmodel.AppBind9{
		Daemon: bind9Daemon,
	}
}
