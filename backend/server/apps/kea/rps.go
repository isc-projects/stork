package kea

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

type RpsPuller struct {
	*agentcomm.PeriodicPuller
	PreviousRps map[int64]StatSample // map of last known values per Daemon
}

// Represents a time/value pair
type StatSample struct {
	SampledAt time.Time // time value was recorded
	Value     int64     // statistic value
}

// Represents a response from the single Kea server to the statistic-get
// for pkt4-ack-sent:
//{
//    "command": "statistic-get",
//    "arguments": {
//        "pkt4-ack-sent": [
//          [ 125, "2019-07-30 10:11:19.498739" ],
//            ...
//          ]
//    },
//    "result": 0
//}
type StatGetResponse4 struct {
	agentcomm.KeaResponseHeader
	Arguments *ResponseArguments4 `json:"arguments,omitempty"`
}

type ResponseArguments4 struct {
	Samples []interface{} `json:"pkt4-ack-sent"`
}

// Represents a response from the single Kea server to the statistic-get
// for pkt6-reply-sent:
//{
//    "command": "statistic-get",
//    "arguments": {
//        "pkt6-reply-sent": [
//          [ 125, "2019-07-30 10:11:19.498739" ],
//            ...
//          ]
//    },
//    "result": 0
//}
type StatGetResponse6 struct {
	agentcomm.KeaResponseHeader
	Arguments *ResponseArguments6 `json:"arguments,omitempty"`
}

type ResponseArguments6 struct {
	Samples []interface{} `json:"pkt6-reply-sent"`
}

// Create a RpsPuller object that in background pulls Kea RPS stats.
// Beneath it spawns a goroutine that pulls stats periodically from Kea apps (that are stored in database).
// For now we tie it to same interval used for Kea lease stats.
func NewRpsPuller(db *pg.DB, agents agentcomm.ConnectedAgents) (*RpsPuller, error) {
	rpsPuller := &RpsPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea RPS Stats", "kea_stats_puller_interval", rpsPuller.pullStats)
	if err != nil {
		return nil, err
	}
	rpsPuller.PeriodicPuller = periodicPuller
	rpsPuller.PreviousRps = map[int64]StatSample{}
	return rpsPuller, nil
}

// Shutdown RpsPuller. It stops goroutine that pulls stats.
func (rpsPuller *RpsPuller) Shutdown() {
	rpsPuller.PeriodicPuller.Shutdown()
}

// Pull stats periodically for all Kea apps which Stork is monitoring. The function returns a number
// of apps for which the stats were successfully pulled and last encountered error.
func (rpsPuller *RpsPuller) pullStats() (int, error) {
	// get list of all kea apps from database
	dbApps, err := dbmodel.GetAppsByType(rpsPuller.Db, dbmodel.AppTypeKea)
	if err != nil {
		return 0, err
	}

	// get stats from each kea app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := rpsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %d: %+v", dbApp.ID, err)
		} else {
			appsOkCnt++
		}
	}

	log.Printf("completed pulling RPS stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))

	// update global statistics in db
	//	err = dbmodel.SetStats(rpsPuller.Db, statsMap)
	//	if err != nil {
	//		lastErr = err
	//	}

	return appsOkCnt, lastErr
}

// Get lease stats from given kea app.
func (rpsPuller *RpsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// prepare URL to CA
	ctrlPoint, err := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		return err
	}

	// issue 2 commands to dhcp daemons at once to get their lease stats for v4 and v6
	cmds := []*agentcomm.KeaCommand{}

	// Iterate over active daemons, adding commands for dhcp4 and dhcp6
	// daemons
	dhcp4Daemons := make(agentcomm.KeaDaemons)
	var dhcp4Daemon *dbmodel.KeaDaemon

	dhcp6Daemons := make(agentcomm.KeaDaemons)
	var dhcp6Daemon *dbmodel.KeaDaemon
	for _, d := range dbApp.Daemons {
		if d.KeaDaemon != nil {
			switch d.Name {
			case "dhcp4":
				dhcp4Daemon = d.KeaDaemon
				dhcp4Daemons["dhcp4"] = true
				cmds = append(cmds, &agentcomm.KeaCommand{
					Command:   "statistic-get",
					Daemons:   &dhcp4Daemons,
					Arguments: &map[string]interface{}{"name": "pkt4-ack-sent"}})
			case "dhcp6":
				dhcp6Daemon = d.KeaDaemon
				dhcp6Daemons["dhcp6"] = true
				cmds = append(cmds, &agentcomm.KeaCommand{
					Command:   "statistic-get",
					Daemons:   &dhcp6Daemons,
					Arguments: &map[string]interface{}{"name": "pkt6-reply-sent"}})
			}
		}
	}

	// if no commands, nothing to do
	if len(cmds) == 0 {
		return nil
	}

	// forward commands to kea
	statsResp4 := []StatGetResponse4{}
	statsResp6 := []StatGetResponse6{}
	ctx := context.Background()
	cmdsResult, err := rpsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, ctrlPoint.Address, ctrlPoint.Port, cmds, &statsResp4, &statsResp6)

	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	if dhcp4Daemon != nil {
		// we should have a result
		if len(statsResp4) == 0 {
			log.Printf("no response to RPS get for KeaDaemon:%d", dhcp4Daemon.DaemonID)
		} else {
			value, sampledAt, err := getSampleRow(statsResp4[0].Arguments.Samples, 0)
			if err != nil {
				log.Printf("could not extract dhcp4 RPS stat: %s", err)
			} else {
				// DO SOMETHING WITH THEM!
				log.Printf("pkt4-ack-sent: KeaDaemon:%d, responses: %d at %+v",
					dhcp4Daemon.DaemonID, value, sampledAt)

				err = rpsPuller.updateDaemonRps(dhcp4Daemon.DaemonID, value, sampledAt)
				if err != nil {
					log.Printf("could not update dhcp4 RPS interval: %s", err)
				}
			}
		}
	}

	if dhcp6Daemon != nil {
		// we should have a result
		if len(statsResp6) == 0 {
			log.Printf("no response to RPS get for KeaDaemon:%d", dhcp6Daemon.DaemonID)
		} else {
			value, sampledAt, err := getSampleRow(statsResp6[0].Arguments.Samples, 0)
			if err != nil {
				log.Printf("could not extract dhcp6 RPS stat: %s", err)
			} else {
				// DO SOMETHING WITH THEM!
				log.Printf("pkt6-reply-sent: KeaDaemon:%d, reponses: %d at %+v",
					dhcp6Daemon.DaemonID, value, sampledAt)

				err = rpsPuller.updateDaemonRps(dhcp6Daemon.DaemonID, value, sampledAt)
				if err != nil {
					log.Printf("could not  update dhcp6 RPS interval: %s", err)
				}
			}
		}
	}

	return nil
}

// todo change this to return timestamp as int64 epoch secs
func getSampleRow(samples []interface{}, rowIdx int) (int64, time.Time, error) {
	// We use current Stork server time rather than Daemon reported sample time
	// so interval times across Daemons are consistent and relative to us. In
	// other words, we don't care when Kea modified the value, we care about
	// when we got it.
	sampledAt := storkutil.UTCNow()

	if samples == nil {
		return 0, sampledAt, errors.New("samples cannot be nil")
	}

	if len(samples) < (rowIdx + 1) {
		// not enough rows
		return 0, sampledAt, fmt.Errorf("sampleList does not a row at idx: %d", rowIdx)
	}

	row, ok := samples[rowIdx].([]interface{})
	if !ok {
		return 0, sampledAt, fmt.Errorf("problem with casting sample row: %+v",
			samples[rowIdx])
	}

	if len(row) != 2 {
		return 0, sampledAt, fmt.Errorf("row has incorrect number of values: %+v", row)
	}

	// not sure unmarshalling makes it a float64.  I guess this is erring on the
	// side of caution?
	value := int64(row[0].(float64))

	return value, sampledAt, nil
}

func (rpsPuller *RpsPuller) updateDaemonRps(daemonID int64, value int64, sampledAt time.Time) error {
	var err error = nil

	// If we have a previous recording, calculate a delta row for it
	if previous, exist := rpsPuller.PreviousRps[daemonID]; exist {
		// Make a new interval
		interval := &dbmodel.RpsInterval{}
		interval.KeaDaemonID = daemonID
		interval.StartTime = previous.SampledAt

		// Calculate the time between the two samples.
		interval.Duration = (sampledAt.Unix() - previous.SampledAt.Unix())

		// Calculate the delta in responses sent.
		if value > previous.Value {
			// New value is larger, we assume we have contiguous data.
			interval.Responses = value - previous.Value
		} else {
			// We have either Kea restart, reset, or statistic rollover. This value
			// then represents the number packets sent since that event occurred.
			interval.Responses = value
		}

		err = dbmodel.AddRpsInterval(rpsPuller.Db, interval)
	}

	// Always update the last reported values for the Daemon.
	rpsPuller.PreviousRps[daemonID] = StatSample{sampledAt, value}

	return err
}
