package bind9

import (
	"context"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/appdata/bind9stats"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// The puller responsible for fetching the statistics from the Bind 9 daemon.
type StatsPuller struct {
	*agentcomm.PeriodicPuller
	EventCenter eventcenter.EventCenter
}

// Create a StatsPuller object that in background pulls BIND 9 statistics.
// Beneath it spawns a goroutine that pulls stats periodically from the BIND 9
// statistics-channel.
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter) (*StatsPuller, error) {
	statsPuller := &StatsPuller{
		EventCenter: eventCenter,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "BIND 9 stats puller", "bind9_stats_puller_interval",
		statsPuller.pullStats)
	if err != nil {
		return nil, err
	}
	statsPuller.PeriodicPuller = periodicPuller
	return statsPuller, nil
}

// Shutdown StatsPuller. It stops goroutine that pulls stats.
func (statsPuller *StatsPuller) Shutdown() {
	statsPuller.PeriodicPuller.Shutdown()
}

// Pull stats periodically for all BIND 9 apps which Stork is monitoring.
// The function returns last encountered error.
func (statsPuller *StatsPuller) pullStats() error {
	// get list of all bind9 apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.DB, dbmodel.AppTypeBind9)
	if err != nil {
		return err
	}

	// get stats from each bind9 app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("Error occurred while getting stats from app %+v: %+v", dbApp, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("Completed pulling stats from BIND 9 apps: %d/%d succeeded", appsOkCnt, len(dbApps))
	return lastErr
}

// Get stats from given bind9 app.
func (statsPuller *StatsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// If the BIND 9 process has been detected but the connection to the
	// daemon cannot be established, then the statistics cannot be pulled.
	// If app or daemon not active then do nothing.
	if len(dbApp.Daemons) == 0 || !dbApp.Daemons[0].Active {
		return nil
	}

	// Prepare URL to statistics-channel.
	statsChannel, err := dbApp.GetAccessPoint(dbmodel.AccessPointStatistics)
	if err != nil {
		return err
	}

	statsOutput := NamedStatsGetResponse{}
	ctx := context.Background()
	err = statsPuller.Agents.ForwardToNamedStats(ctx, dbApp, statsChannel.Address, statsChannel.Port, "", &statsOutput)
	if err != nil {
		return err
	}

	namedStats := &bind9stats.Bind9NamedStats{}

	if statsOutput.Views != nil {
		viewStats := make(map[string]*bind9stats.Bind9StatsView)

		for name, view := range statsOutput.Views {
			// Exclude _bind view as it is a special kind of view for which
			// we don't have query stats.
			if name == "_bind" {
				continue
			}

			cacheStats := make(map[string]int64)
			cacheStats["CacheHits"] = view.Resolver.CacheStats.CacheHits
			cacheStats["CacheMisses"] = view.Resolver.CacheStats.CacheMisses
			cacheStats["QueryHits"] = view.Resolver.CacheStats.QueryHits
			cacheStats["QueryMisses"] = view.Resolver.CacheStats.QueryMisses

			viewStats[name] = &bind9stats.Bind9StatsView{
				Resolver: &bind9stats.Bind9StatsResolver{
					CacheStats: cacheStats,
				},
			}
		}

		namedStats.Views = viewStats
	}

	dbApp.Daemons[0].Bind9Daemon.Stats.NamedStats = namedStats
	return dbmodel.UpdateDaemon(statsPuller.DB, dbApp.Daemons[0])
}
