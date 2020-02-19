package kea

import (
	"context"
	"sync"
	"time"

	"github.com/go-pg/pg/v9"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

type StatsPuller struct {
	Db     *pg.DB
	Agents agentcomm.ConnectedAgents
	Ticker *time.Ticker
	Done   chan bool
	Wg     *sync.WaitGroup
}

// Create a StatsPuller object. Beneath it spawns a goroutine that pulls stats
// periodically from Kea apps (that are stored in database).
// pullingInterval argument is expressed in seconds.
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents) *StatsPuller {
	log.Printf("Starting Stats Puller")
	statsPuller := &StatsPuller{
		Db:     db,
		Agents: agents,
		Ticker: time.NewTicker(10 * time.Minute), // TODO: change it to a setting in db
		Done:   make(chan bool),
		Wg:     &sync.WaitGroup{},
	}

	// start puller loop as goroutine and increment WaitGroup (which is used later
	// for stopping this goroutine)
	statsPuller.Wg.Add(1)
	go statsPuller.pullerLoop()

	log.Printf("Started Stats Puller")
	return statsPuller
}

// Shutdown StatsPuller. It stops goroutine that pulls stats.
func (statsPuller *StatsPuller) Shutdown() {
	log.Printf("Stopping Kea Stats Puller")
	statsPuller.Ticker.Stop()
	statsPuller.Done <- true
	statsPuller.Wg.Wait()
	log.Printf("Stopped Kea Stats Puller")
}

// A loop that pulls stats from all Kea apps. It pulls periodicaly by indicated time
// in configuration.
func (statsPuller *StatsPuller) pullerLoop() {
	defer statsPuller.Wg.Done()
	for {
		select {
		// every N seconds do lease stats gathering from all kea apps and their active daemons
		case <-statsPuller.Ticker.C:
			_, err := statsPuller.pullLeaseStats()
			if err != nil {
				log.Errorf("some errors were encountered while gathering lease stats from Kea apps: %+v", err)
			}
		// wait for done signal from shutdown function
		case <-statsPuller.Done:
			return
		}
	}
}

// Pull stats from all Kea apps from database. It returns number of successfuly pulled apps
// and last encountered error.
func (statsPuller *StatsPuller) pullLeaseStats() (int, error) {
	// get list of all kea apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.Db, dbmodel.KeaAppType)
	if err != nil {
		return 0, err
	}

	// get lease stats from each kea app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getLeaseStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %+v: %+v", dbApp, err)
		} else {
			appsOkCnt++
		}
	}
	return appsOkCnt, lastErr
}

// Part of response for stat-lease4-get and stat-lease6-get commands.
type ResultSetInStatLeaseGet struct {
	Columns []string
	Rows    [][]int
}

// Part of response for stat-lease4-get and stat-lease6-get commands.
type StatLeaseGetArgs struct {
	ResultSet ResultSetInStatLeaseGet `json:"result-set"`
	Timestamp string
}

// Represents unmarshaled response from Kea daemon to stat-lease4-get and stat-lease6-get commands.
type StatLeaseGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *StatLeaseGetArgs `json:"arguments,omitempty"`
}

// Get lease stats from given kea app.
func (statsPuller *StatsPuller) getLeaseStatsFromApp(dbApp *dbmodel.App) error {
	// prepare URL to CA
	caURL := storkutil.HostWithPortURL(dbApp.CtrlAddress, dbApp.CtrlPort)

	// get active dhcp daemons
	dhcpDaemons := make(agentcomm.KeaDaemons)
	found := false
	for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
		if d.Name == "dhcp4" || d.Name == "dhcp6" {
			dhcpDaemons[d.Name] = true
			found = true
		}
	}
	// if no dhcp daemons found then exit
	if !found {
		return nil
	}

	// issue 2 commands to dhcp daemons at once to get their lease stats for v4 and v6
	cmds := []*agentcomm.KeaCommand{}
	if dhcpDaemons["dhcp4"] {
		cmds = append(cmds, &agentcomm.KeaCommand{
			Command: "stat-lease4-get",
			Daemons: &dhcpDaemons,
		})
	}
	if dhcpDaemons["dhcp6"] {
		cmds = append(cmds, &agentcomm.KeaCommand{
			Command: "stat-lease6-get",
			Daemons: &dhcpDaemons,
		})
	}

	// forward commands to kea
	stats4Resp := []StatLeaseGetResponse{}
	stats6Resp := []StatLeaseGetResponse{}
	ctx := context.Background()
	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, caURL, cmds, &stats4Resp, &stats6Resp)
	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	// process response from kea daemons
	log.Printf("App %+v", dbApp)
	log.Printf("stats4Resp %+v", stats4Resp)
	for _, s4r := range stats4Resp {
		if s4r.Arguments == nil {
			continue
		}
		for idx, row := range s4r.Arguments.ResultSet.Rows {
			log.Printf("Row: %d", idx)
			for colIdx, col := range row {
				log.Printf("  %s: %d", s4r.Arguments.ResultSet.Columns[colIdx], col)
			}
		}
	}
	log.Printf("stats6Resp %+v", stats6Resp)
	for _, s6r := range stats6Resp {
		if s6r.Arguments == nil {
			continue
		}
		for idx, row := range s6r.Arguments.ResultSet.Rows {
			log.Printf("Row: %d", idx)
			for colIdx, col := range row {
				log.Printf("  %s: %d", s6r.Arguments.ResultSet.Columns[colIdx], col)
			}
		}
	}

	return nil
}
