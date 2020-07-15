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

// Periodic Puller that generates RPS interval data.
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

// The list of value/timestamp pairs returned as pkt4-ack-sent
// as the value for command response "Arguments" element
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

// The list of value/timestamp pairs returned as pkt6-reply-sent
// as the value for command response "Arguments" element
type ResponseArguments6 struct {
	Samples []interface{} `json:"pkt6-reply-sent"`
}

// Create a RpsPuller object that in background pulls Kea RPS stats.
// Beneath it spawns a goroutine that pulls the response sent statistics
// periodically from Kea apps (that are stored in database).  These are
// used to calculate and store RPS interval data per Kea daemon in the database.
// For now we tie it to same pull interval value used for Kea lease stats.
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

// Pull RPS stats periodically for all Kea apps which Stork is monitoring. The function
// returns the number of apps for which the stats were successfully pulled and last
// encountered error.
func (rpsPuller *RpsPuller) pullStats() (int, error) {
	// Age off records more than 24 hrs 30 min old
	deleteTime := storkutil.UTCNow().Add(time.Duration(-88200) * time.Second)
	err := dbmodel.AgeOffRpsInterval(rpsPuller.Db, deleteTime)
	if err != nil {
		return 0, err
	}

	// Get list of all kea apps from database
	dbApps, err := dbmodel.GetAppsByType(rpsPuller.Db, dbmodel.AppTypeKea)
	if err != nil {
		return 0, err
	}

	// Get RPS stats from each kea app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := rpsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting RPS stats from app %d: %+v", dbApp.ID, err)
		} else {
			appsOkCnt++
		}
	}

	log.Printf("completed pulling RPS stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))

	return appsOkCnt, lastErr
}

// Generates RPS interval data for each daemon in a given Kea app
func (rpsPuller *RpsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// Prepare URL to CA
	ctrlPoint, err := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		return err
	}

	// Issue two commands to dhcp daemons at once to get their lease stats for v4 and v6
	cmds := []*agentcomm.KeaCommand{}

	// Iterate over active daemons, adding commands for dhcp4 and dhcp6 daemons
	dhcp4Daemons := make(agentcomm.KeaDaemons)
	var dhcp4Daemon *dbmodel.KeaDaemon
	var dhcp4Arguments = RpsGetDhcp4Arguments()

	dhcp6Daemons := make(agentcomm.KeaDaemons)
	var dhcp6Daemon *dbmodel.KeaDaemon
	var dhcp6Arguments = RpsGetDhcp6Arguments()

	// Since we might have dhcp4 only, dhcp6 only or both, we build an array
	// of expected responses.
	var responses [2]interface{}
	rspIdx := 0
	for _, d := range dbApp.Daemons {
		if d.KeaDaemon != nil && d.Active {
			switch d.Name {
			case "dhcp4":
				dhcp4Daemon = d.KeaDaemon
				dhcp4Daemons["dhcp4"] = true
				cmds = append(cmds, &agentcomm.KeaCommand{
					Command:   "statistic-get",
					Daemons:   &dhcp4Daemons,
					Arguments: &dhcp4Arguments})
				// fill in the response type
				responses[rspIdx] = &[]StatGetResponse4{}
				rspIdx++
			case "dhcp6":
				dhcp6Daemon = d.KeaDaemon
				dhcp6Daemons["dhcp6"] = true
				cmds = append(cmds, &agentcomm.KeaCommand{
					Command:   "statistic-get",
					Daemons:   &dhcp6Daemons,
					Arguments: &dhcp6Arguments})
				responses[rspIdx] = &[]StatGetResponse6{}
			}
		}
	}

	// If there are no commands, nothing to do
	if len(cmds) == 0 {
		return nil
	}

	for i := 0; i < len(cmds); i++ {
		log.Printf("TKM rpsPuller cmds: %d: %+v", i, cmds[i].Command)
		log.Printf("TKM rpsPuller cmds: %d: %+v", i, cmds[i].Daemons)
		log.Printf("TKM rpsPuller cmds: %d: %+v", i, cmds[i].Arguments)
	}

	// forward commands to kea
	ctx := context.Background()

	cmdsResult, err := rpsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, ctrlPoint.Address, ctrlPoint.Port, cmds, responses[0], responses[1])

	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	rspIdx = 0
	// If we have a daemon, we should have a result
	if dhcp4Daemon != nil {
		statsResp4, ok := responses[rspIdx].(*[]StatGetResponse4)
		if !ok {
			log.Printf("response type is invalid: %+v", responses[rspIdx])
		} else {
			samples, err := rpsPuller.extractSamples4(dhcp4Daemon.DaemonID, *statsResp4)
			if err == nil {
				// Calculate and store the RPS interval for this daemon for this cycle
				err = rpsPuller.updateDaemonRps(dhcp4Daemon.DaemonID, samples)
			}

			if err != nil {
				log.Printf("could not update dhcp4 RPS interval: %s", err)
			}
		}

		// Bump the response index
		rspIdx++
	}

	// If we have a daemon, we should have a result
	if dhcp6Daemon != nil {
		statsResp6, ok := responses[rspIdx].(*[]StatGetResponse6)
		if !ok {
			log.Printf("response type is invalid: %+v", responses[rspIdx])
		} else {
			samples, err := rpsPuller.extractSamples6(dhcp6Daemon.DaemonID, *statsResp6)
			if err == nil {
				// Calculate and store the RPS interval for this daemon for this cycle
				err = rpsPuller.updateDaemonRps(dhcp6Daemon.DaemonID, samples)
			}

			if err != nil {
				log.Printf("could not update dhcp6 RPS interval: %s", err)
			}
		}
	}

	return nil
}

// Exract the list of statistic samples from a dhcp4 statistic-get response if the response is valid.
func (rpsPuller *RpsPuller) extractSamples4(daemonID int64, statsResp []StatGetResponse4) ([]interface{}, error) {
	if len(statsResp) == 0 {
		err := fmt.Errorf("empty RPS response for KeaDaemon:%d", daemonID)
		return nil, err
	}

	if statsResp[0].Result != 0 {
		err := fmt.Errorf("error result in RPS response for for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	if statsResp[0].Arguments == nil {
		err := fmt.Errorf("missing Arguments from RPS response for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	if statsResp[0].Arguments.Samples == nil {
		err := fmt.Errorf("missing Samples from RPS response for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	return statsResp[0].Arguments.Samples, nil
}

// Exract the list of statistic samples from a dhcp6 statistic-get response if the response is valid.
func (rpsPuller *RpsPuller) extractSamples6(daemonID int64, statsResp []StatGetResponse6) ([]interface{}, error) {
	if len(statsResp) == 0 {
		err := fmt.Errorf("empty RPS response for KeaDaemon:%d", daemonID)
		return nil, err
	}

	if statsResp[0].Result != 0 {
		err := fmt.Errorf("error result in RPS response for for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	if statsResp[0].Arguments == nil {
		err := fmt.Errorf("missing Arguments from RPS response for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	if statsResp[0].Arguments.Samples == nil {
		err := fmt.Errorf("missing Samples from RPS response for KeaDaemon:%d, response: %+v", daemonID, statsResp)
		return nil, err
	}

	return statsResp[0].Arguments.Samples, nil
}

// Uses the most recent Kea statistic value for packets sent to calculate and
// store an RPS interval row for the current interval for the given daemon.
func (rpsPuller *RpsPuller) updateDaemonRps(daemonID int64, samples []interface{}) error {
	// The first row of the samples is the most recent value and the only
	// one we care about. Fetch it.
	value, sampledAt, err := getSampleRow(samples, 0)
	if err != nil {
		return fmt.Errorf("could not extract RPS stat: %s", err)
	}

	if value < 0 {
		// Shouldn't happen but if it does, we'll record a 0.
		log.Warnf("discarding response value: %d returned from KeaDaemonID: %d", value, daemonID)
		value = int64(0)
	}

	// If we have a previous recording, calculate a delta row for it
	if previous, exist := rpsPuller.PreviousRps[daemonID]; exist {
		// Make a new interval
		interval := &dbmodel.RpsInterval{}
		interval.KeaDaemonID = daemonID
		interval.StartTime = previous.SampledAt

		// Calculate the time between the two samples.
		interval.Duration = (sampledAt.Unix() - previous.SampledAt.Unix())

		// Calculate the delta in responses sent.
		if value >= previous.Value {
			// New value is larger, we assume we have contiguous data.
			interval.Responses = value - previous.Value
		} else {
			// We have either Kea restart, reset, or statistic rollover. This value
			// then represents the number packets sent since that event occurred.
			interval.Responses = value
		}

		log.Printf("TKM insert interval: %+v", interval)
		err = dbmodel.AddRpsInterval(rpsPuller.Db, interval)
	}

	// Always update the last reported values for the Daemon.
	rpsPuller.PreviousRps[daemonID] = StatSample{sampledAt, value}

	return err
}

// Returns the statistic value and sample time from a given row within a
// a list of samples.  Note that rather than use the sample time in the list,
// We use current Stork server time so interval times across Daemons are
// consistent and relative to us. In other words, we don't care when Kea
// modified the value, we care about when we got it.
func getSampleRow(samples []interface{}, rowIdx int) (int64, time.Time, error) {
	sampledAt := storkutil.UTCNow()
	if samples == nil {
		return 0, sampledAt, errors.New("samples cannot be nil")
	}

	if len(samples) < (rowIdx + 1) {
		// Not enough rows
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

	// Not sure why unmarshalling makes it a float64, but we need an int64.
	log.Printf("TKM value before cast: %+v", row[0].(float64))

	value := int64(row[0].(float64))

	return value, sampledAt, nil
}

// "Static" constant for dhcp4 statistic-get command argument
func RpsGetDhcp4Arguments() map[string]interface{} {
	return map[string]interface{}{"name": "pkt4-ack-sent"}
}

// "Static" constant for dhcp6 statistic-get command argument
func RpsGetDhcp6Arguments() map[string]interface{} {
	return map[string]interface{}{"name": "pkt6-reply-sent"}
}
