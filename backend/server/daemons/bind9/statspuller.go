package bind9

import (
	"context"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/datamodel/daemonname"
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

// Pull stats periodically for all BIND 9 daemons which Stork is monitoring.
// The function returns last encountered error.
func (statsPuller *StatsPuller) pullStats() error {
	// get list of all bind9 daemons from database
	daemons, err := dbmodel.GetDaemonsByName(statsPuller.DB, daemonname.Bind9)
	if err != nil {
		return err
	}

	// get stats from each bind9 daemon
	var lastErr error
	okCnt := 0
	for _, daemon := range daemons {
		err := statsPuller.getStatsFromDaemon(&daemon)
		if err != nil {
			lastErr = err
			log.WithError(err).Errorf("Error occurred while getting stats from daemon %+v", daemon)
		} else {
			okCnt++
		}
	}
	log.Printf("Completed pulling stats from BIND 9 daemons: %d/%d succeeded", okCnt, len(daemons))
	return lastErr
}

// Get stats from given bind9 daemon.
func (statsPuller *StatsPuller) getStatsFromDaemon(daemon *dbmodel.Daemon) error {
	// If the BIND 9 process has been detected but the connection to the
	// daemon cannot be established, then the statistics cannot be pulled.
	// If daemon is not active then do nothing.
	if !daemon.Active {
		return nil
	}

	err := GetDaemonStatistics(context.Background(), statsPuller.Agents, daemon)
	if err != nil {
		return err
	}
	return dbmodel.UpdateDaemonStatistics(statsPuller.DB, daemon)
}
