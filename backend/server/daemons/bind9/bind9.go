package bind9

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	agentapi "isc.org/stork/api"
	"isc.org/stork/daemondata/bind9stats"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Provide example date format how named returns dates.
const namedLongDateFormat = "Mon, 02 Jan 2006 15:04:05 MST"

// The cache statistics of the Bind9 named daemon.
type CacheStatsData struct {
	CacheHits   int64 `json:"CacheHits"`
	CacheMisses int64 `json:"CacheMisses"`
	QueryHits   int64 `json:"QueryHits"`
	QueryMisses int64 `json:"QueryMisses"`
}

// The resolver entry of the view statistics JSON structure.
type ResolverData struct {
	CacheStats CacheStatsData `json:"cachestats"`
}

// The view statistics data JSON structure.
type ViewStatsData struct {
	Resolver ResolverData `json:"resolver"`
}

// JSON Structure of response returned by the named Bind 9 daemon on fetching
// statistics.
type NamedStatsGetResponse struct {
	Views map[string]*ViewStatsData `json:"views,omitempty"`
}

// Get statistics from named daemon using ForwardToNamedStats function.
func GetDaemonStatistics(ctx context.Context, agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon) error {
	// prepare URL to named
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// store all collected details in daemon db record
	statsOutput := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx2, daemon, agentapi.ForwardToNamedStatsReq_SERVER, &statsOutput)
	if err != nil {
		return errors.WithMessage(err, "problem retrieving stats from named")
	}

	if statsOutput.Views == nil {
		// Nothing to do.
		return nil
	}

	namedStats := bind9stats.Bind9NamedStats{}

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

		namedStats.Views = viewStats
	}

	daemon.Bind9Daemon.Stats.NamedStats = namedStats
	return nil
}

// Get state of named daemon using ForwardRndcCommand function.
// The state that is stored into daemon includes: version, number of zones, and
// some runtime state.
func GetDaemonState(ctx context.Context, agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon, eventCenter eventcenter.EventCenter) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	command := "status"
	out, err := agents.ForwardRndcCommand(ctx2, daemon, command)
	if err != nil {
		log.Warnf("Problem getting BIND 9 status: %s", err)
		return
	}

	// Get version
	pattern := regexp.MustCompile(`version:\s+(.+)\n`)
	match := pattern.FindStringSubmatch(out.Output)
	if match != nil {
		daemon.Version = match[1]
	} else {
		log.Warnf("Cannot get BIND 9 version: unable to find version in output")
	}

	// Is the named daemon running?
	pattern = regexp.MustCompile(`server is up and running`)
	up := pattern.FindString(out.Output)
	daemon.Active = up != ""

	// Up time
	pattern = regexp.MustCompile(`boot time:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		bootTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("Cannot get BIND 9 uptime: %s", err.Error())
		}
		now := time.Now()
		elapsed := now.Sub(bootTime)
		daemon.Uptime = int64(elapsed.Seconds())
	} else {
		log.Warnf("Cannot get BIND 9 uptime: unable to find boot time in output")
	}

	// Reloaded at
	pattern = regexp.MustCompile(`last configured:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		reloadTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("Cannot get BIND 9 reload time: %s", err.Error())
		}
		daemon.ReloadedAt = reloadTime
	} else {
		log.Warnf("Cannot get BIND 9 reload time: unable to find last configured time in output")
	}

	// Number of zones
	pattern = regexp.MustCompile(`number of zones:\s+(\d+)\s+\((\d+) automatic\)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		count, err := strconv.Atoi(match[1])
		if err != nil {
			log.Warnf("Cannot get BIND 9 number of zones: %s", err.Error())
		}
		autoCount, err := strconv.Atoi(match[2])
		if err != nil {
			log.Warnf("Cannot get BIND 9 number of automatic zones: %s", err.Error())
		}
		daemon.Bind9Daemon.Stats.ZoneCount = int64(count - autoCount)
		daemon.Bind9Daemon.Stats.AutomaticZoneCount = int64(autoCount)
	} else {
		log.Warnf("Cannot get BIND 9 number of zones: unable to find number of zones in output")
	}

	// Get statistics
	err = GetDaemonStatistics(ctx, agents, daemon)
	if err != nil {
		log.Warnf("Problem getting BIND 9 statistics: %s", err)
	}
}

// Inserts or updates information about BIND 9 daemon in the database.
func CommitDaemonIntoDB(db *dbops.PgDB, daemon *dbmodel.Daemon, eventCenter eventcenter.EventCenter) (err error) {
	if daemon.ID == 0 {
		err = dbmodel.AddDaemon(db, daemon)
		eventCenter.AddInfoEvent("added {daemon}", daemon.Machine, daemon)
	} else {
		err = dbmodel.UpdateDaemon(db, daemon)
	}
	// todo: perform any additional actions required after storing the
	// daemon in the db.
	return err
}
