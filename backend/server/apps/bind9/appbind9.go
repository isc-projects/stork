package bind9

import (
	"context"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Provide example date format how named returns dates.
const namedLongDateFormat = "Mon, 02 Jan 2006 15:04:05 MST"

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
		Pid:  0,
		Name: "named",
	}

	// Get version
	pattern := regexp.MustCompile(`version:\s+(.+)\n`)
	match := pattern.FindStringSubmatch(out.Output)
	if match != nil {
		bind9Daemon.Version = match[1]
	} else {
		log.Warnf("cannot get BIND 9 version: unable to find version in output")
	}

	// Is the named daemon running?
	bind9Daemon.Active = false
	pattern = regexp.MustCompile(`server is up and running`)
	up := pattern.FindString(out.Output)
	if up != "" {
		bind9Daemon.Active = true
	}

	// Up time
	pattern = regexp.MustCompile(`boot time:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		bootTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 up time: %s", err.Error())
		}
		now := time.Now()
		elapsed := now.Sub(bootTime)
		bind9Daemon.Uptime = int64(elapsed.Seconds())
	} else {
		log.Warnf("cannot get BIND 9 up time: unable to find boot time in output")
	}

	// Reloaded at
	pattern = regexp.MustCompile(`last configured:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		reloadTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 reload time: %s", err.Error())
		}
		bind9Daemon.ReloadedAt = reloadTime
	} else {
		log.Warnf("cannot get BIND 9 reload time: unable to find last configured in output")
	}

	// Number of zones
	pattern = regexp.MustCompile(`number of zones:\s+(\d+)\s+\((\d+) automatic\)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		count, err := strconv.Atoi(match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 number of zones: %s", err.Error())
		}
		autoCount, err := strconv.Atoi(match[2])
		if err != nil {
			log.Warnf("cannot get BIND 9 number of automatic zones: %s", err.Error())
		}
		bind9Daemon.ZoneCount = int64(count - autoCount)
		bind9Daemon.AutomaticZoneCount = int64(autoCount)
	} else {
		log.Warnf("cannot get BIND 9 number of zones: unable to find number of zones in output")
	}

	// Save status
	dbApp.Active = bind9Daemon.Active
	dbApp.Meta.Version = bind9Daemon.Version
	dbApp.Details = dbmodel.AppBind9{
		Daemon: bind9Daemon,
	}
}

// Inserts or updates information about BIND 9 app in the database.
func CommitAppIntoDB(db *dbops.PgDB, app *dbmodel.App) (err error) {
	if app.ID == 0 {
		err = dbmodel.AddApp(db, app)
	} else {
		err = dbmodel.UpdateApp(db, app)
	}
	// todo: perform any additional actions required after storing the
	// app in the db.
	return err
}
