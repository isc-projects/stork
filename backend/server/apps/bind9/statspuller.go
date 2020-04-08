package bind9

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v9"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

type StatsPuller struct {
	*agentcomm.PeriodicPuller
}

// Create a StatsPuller object that in background pulls BIND 9 statistics.
// Beneath it spawns a goroutine that pulls stats periodically from the BIND 9
// statistics-channel.
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents) (*StatsPuller, error) {
	statsPuller := &StatsPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "BIND 9 Stats", "bind9_stats_puller_interval",
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
// The function returns a number of apps for which the stats were successfully
// pulled and last encountered error.
func (statsPuller *StatsPuller) pullStats() (int, error) {
	// get list of all bind9 apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.Db, dbmodel.AppTypeBind9)
	if err != nil {
		return 0, err
	}

	// get stats from each bind9 app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %+v: %+v", dbApp, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("completed pulling stats from BIND 9 apps: %d/%d succeeded", appsOkCnt, len(dbApps))
	return appsOkCnt, lastErr
}

// Get stats from given bind9 app.
func (statsPuller *StatsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// prepare URL to statistics-channel
	statsChannel, err := dbApp.GetAccessPoint(dbmodel.AccessPointStatistics)
	if err != nil {
		return err
	}
	statsAddress := storkutil.HostWithPortURL(statsChannel.Address, statsChannel.Port)
	statsRequest := "json/v1/server"
	statsURL := fmt.Sprintf("%s%s", statsAddress, statsRequest)

	statsOutput := NamedStatsGetResponse{}
	ctx := context.Background()
	err = statsPuller.Agents.ForwardToNamedStats(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, statsURL, &statsOutput)
	if err != nil {
		return err
	}

	bind9App := dbApp.Details.(dbmodel.AppBind9)
	bind9Daemon := bind9App.Daemon

	if statsOutput.Views != nil {
		for name, view := range statsOutput.Views {
			// Only deal with the default view for now.
			if name != "_default" {
				continue
			}
			// Calculate the cache hit ratio: the number of
			// responses that were retrieved from cache divided
			// by the number of all responses.
			hits := view.Resolver.CacheStats.CacheHits
			misses := view.Resolver.CacheStats.CacheMisses
			ratio := float64(0)
			total := float64(hits) + float64(misses)
			if total != 0 {
				ratio = float64(hits) / total
			}
			bind9Daemon.CacheHitRatio = ratio
			bind9Daemon.CacheHits = hits
			bind9Daemon.CacheMisses = misses
			break
		}
	}

	dbApp.Details = dbmodel.AppBind9{
		Daemon: bind9Daemon,
	}

	return CommitAppIntoDB(statsPuller.Db, dbApp)
}
