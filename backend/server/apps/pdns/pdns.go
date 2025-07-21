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
// provided app instance.
func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, eventCenter eventcenter.EventCenter) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try to get the server info from the agent.
	serverInfo, err := agents.GetPowerDNSServerInfo(ctx2, dbApp)
	if err != nil {
		log.Warnf("Problem getting PowerDNS server info: %s", err)
		return
	}

	// Check if the app already contains a daemon.
	var daemon *dbmodel.Daemon
	if len(dbApp.Daemons) > 0 && dbApp.Daemons[0].ID != 0 {
		daemon = dbApp.Daemons[0]
	} else {
		// This is the first time we see this app, so it has no daemon.
		daemon = dbmodel.NewPDNSDaemon(true)
	}
	daemon.Version = serverInfo.Version
	daemon.ExtendedVersion = serverInfo.Version
	daemon.Uptime = serverInfo.Uptime
	daemon.PDNSDaemon.Details.URL = serverInfo.URL
	daemon.PDNSDaemon.Details.ConfigURL = serverInfo.ConfigURL
	daemon.PDNSDaemon.Details.ZonesURL = serverInfo.ZonesURL
	daemon.PDNSDaemon.Details.AutoprimariesURL = serverInfo.AutoprimariesURL

	dbApp.Active = daemon.Active
	dbApp.Meta.Version = daemon.Version
	dbApp.Meta.ExtendedVersion = daemon.ExtendedVersion
	dbApp.Daemons = []*dbmodel.Daemon{
		daemon,
	}
}

// Inserts or updates information about PowerDNS app in the database.
func CommitAppIntoDB(db *pg.DB, app *dbmodel.App, eventCenter eventcenter.EventCenter) (err error) {
	if app.ID == 0 {
		_, err = dbmodel.AddApp(db, app)
		eventCenter.AddInfoEvent("added {app}", app.Machine, app)
	} else {
		_, _, err = dbmodel.UpdateApp(db, app)
	}
	return err
}
