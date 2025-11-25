package pdns

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Fetches the general information about the PowerDNS server and updates the
// provided daemon instance.
func GetDaemonState(ctx context.Context, agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon, eventCenter eventcenter.EventCenter) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try to get the server info from the agent.
	serverInfo, err := agents.GetPowerDNSServerInfo(ctx, daemon)
	if err != nil {
		log.WithError(err).Warn("Problem getting PowerDNS server info")
		return
	}
	daemon.Version = serverInfo.Version
	daemon.ExtendedVersion = serverInfo.Version
	daemon.Uptime = serverInfo.Uptime
	daemon.PDNSDaemon.Details.URL = serverInfo.URL
	daemon.PDNSDaemon.Details.ConfigURL = serverInfo.ConfigURL
	daemon.PDNSDaemon.Details.ZonesURL = serverInfo.ZonesURL
	daemon.PDNSDaemon.Details.AutoprimariesURL = serverInfo.AutoprimariesURL
}

// Inserts or updates information about PowerDNS daemon in the database.
func CommitDaemonIntoDB(db *pg.DB, daemon *dbmodel.Daemon, eventCenter eventcenter.EventCenter) (err error) {
	if daemon.ID == 0 {
		err = dbmodel.AddDaemon(db, daemon)
		eventCenter.AddInfoEvent("added {daemon} to {machine}", daemon, daemon.Machine)
	} else {
		err = dbmodel.UpdateDaemon(db, daemon)
	}
	return err
}
