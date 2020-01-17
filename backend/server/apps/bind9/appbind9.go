package bind9

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	state, err := agents.GetBind9State(ctx2, dbApp.Machine.Address, dbApp.Machine.AgentPort)
	if err != nil {
		log.Warnf("problem with getting BIND 9 state: %s", err)
		return
	}

	// store all collected details in app db record
	dbApp.Active = state.Active
	dbApp.Meta.Version = state.Version
	dbApp.Details = dbmodel.AppBind9{
		Daemon: dbmodel.Bind9Daemon{
			Pid:     state.Daemon.Pid,
			Name:    state.Daemon.Name,
			Active:  state.Daemon.Active,
			Version: state.Daemon.Version,
		},
	}
}
