package kea

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keactrl "isc.org/stork/appctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Periodic Puller that generates RPS interval data.
type RpsWorker struct {
	db          *pg.DB
	PreviousRps map[int64]StatSample // map of last known values per Daemon
	Interval1   time.Duration
	Interval2   time.Duration
}

// Represents a time/value pair.
type StatSample struct {
	SampledAt time.Time // time value was recorded
	Value     int64     // statistic value
}

// Create a RpsWorker object for building Kea API commands and using
// their responses to populate RPS statistics.
func NewRpsWorker(db *pg.DB) (*RpsWorker, error) {
	rpsWorker := &RpsWorker{}

	rpsWorker.db = db
	rpsWorker.PreviousRps = map[int64]StatSample{}

	// The interval values may some day be configurable
	rpsWorker.Interval1 = (time.Minute * 15)
	rpsWorker.Interval2 = (time.Hour * 24)

	return rpsWorker, nil
}

// Ages off obsolete RPS interval data.
func (rpsWorker *RpsWorker) AgeOffRpsIntervals() error {
	// Age off records more than Interval2 old.
	deleteTime := storkutil.UTCNow().Add(-rpsWorker.Interval2)
	err := dbmodel.AgeOffRpsInterval(rpsWorker.db, deleteTime)
	return err
}

// Updates the statistic-get-all command response for DHCP4.
func (rpsWorker *RpsWorker) Response4Handler(daemon *dbmodel.Daemon, response keactrl.GetAllStatisticsResponse) error {
	samples, err := rpsWorker.extractSamples4(response)
	if err == nil && samples != nil {
		// Note that rather than use the sample time in the list,
		// We use current Stork Server time so interval times across Daemons are
		// consistent and relative to us. In other words, we don't care when Kea
		// modified the value, we care about when we got it.
		sampledAt := storkutil.UTCNow()

		// Calculate and store the RPS interval for this daemon for this cycle
		err = rpsWorker.updateDaemonRpsIntervals(daemon, samples.Value.Int64(), sampledAt)

		// Now we'll update the Kea RPS statistics based on the updated interval data
		if err == nil {
			err = rpsWorker.updateKeaDaemonRpsStats(daemon)
		}
	}

	if err != nil {
		return errors.WithMessagef(err, "could not update dhcp4 RPS data for %+v", daemon)
	}

	return nil
}

// Processes the statistic-get command response for DHCP4.
func (rpsWorker *RpsWorker) Response6Handler(daemon *dbmodel.Daemon, response keactrl.GetAllStatisticsResponse) error {
	sample, err := rpsWorker.extractSamples6(response)
	if err == nil && sample != nil {
		// Note that rather than use the sample time in the list,
		// We use current Stork Server time so interval times across Daemons are
		// consistent and relative to us. In other words, we don't care when Kea
		// modified the value, we care about when we got it.
		sampledAt := storkutil.UTCNow()

		// Calculate and store the RPS interval for this daemon for this cycle
		err = rpsWorker.updateDaemonRpsIntervals(daemon, sample.Value.Int64(), sampledAt)

		// Now we'll update the Kea RPS statistics based on the updated interval data
		if err == nil {
			err = rpsWorker.updateKeaDaemonRpsStats(daemon)
		}
	}

	if err != nil {
		return errors.WithMessagef(err, "could not update dhcp6 RPS data for %+v", daemon)
	}

	return nil
}

// Extract the list of statistic samples from a dhcp4 statistic-get-all response if the response is valid.
func (rpsWorker *RpsWorker) extractSamples4(statsResp keactrl.GetAllStatisticsResponse) (*keactrl.GetAllStatisticResponseSample, error) {
	if len(statsResp) == 0 {
		err := errors.Errorf("empty RPS response")
		return nil, err
	}

	if err := statsResp[0].GetError(); err != nil {
		err := errors.WithMessage(err, "error result in RPS response")
		return nil, err
	}

	if statsResp[0].Arguments == nil {
		err := errors.Errorf("missing arguments from RPS response %+v", statsResp)
		return nil, err
	}

	for _, sample := range statsResp[0].Arguments {
		if sample.Name == "pkt4-ack-sent" {
			return &sample, nil
		}
	}
	return nil, nil
}

// Extract the list of statistic samples from a dhcp6 statistic-get response if the response is valid.
func (rpsWorker *RpsWorker) extractSamples6(statsResp keactrl.GetAllStatisticsResponse) (*keactrl.GetAllStatisticResponseSample, error) {
	if len(statsResp) == 0 {
		err := errors.Errorf("empty RPS response")
		return nil, err
	}

	if err := statsResp[0].GetError(); err != nil {
		err := errors.WithMessage(err, "error result in RPS response")
		return nil, err
	}

	if statsResp[0].Arguments == nil {
		err := errors.Errorf("missing arguments from RPS response: %+v", statsResp)
		return nil, err
	}

	if statsResp[0].Arguments == nil {
		err := errors.Errorf("missing samples from RPS response: %+v", statsResp)
		return nil, err
	}

	for _, sample := range statsResp[0].Arguments {
		if sample.Name == "pkt6-reply-sent" {
			return &sample, nil
		}
	}
	return nil, nil
}

// Uses the most recent Kea statistic value for packets sent to calculate and
// store an RPS interval row for the current interval for the given daemon.
func (rpsWorker *RpsWorker) updateDaemonRpsIntervals(daemon *dbmodel.Daemon, value int64, timestamp time.Time) (err error) {
	// The first row of the samples is the most recent value and the only
	// one we care about. Fetch it.
	daemonID := daemon.KeaDaemon.DaemonID
	if value < 0 {
		// Shouldn't happen but if it does, we'll record a 0.
		log.Warnf("Discarding response value: %d returned from KeaDaemonID: %d", value, daemonID)
		value = int64(0)
	}

	// If we have a previous recording, calculate a delta row for it
	if previous, exist := rpsWorker.PreviousRps[daemonID]; exist {
		// Make a new interval
		interval := &dbmodel.RpsInterval{}
		interval.KeaDaemonID = daemonID
		interval.StartTime = previous.SampledAt

		// Calculate the time between the two samples.
		interval.Duration = (timestamp.Unix() - previous.SampledAt.Unix())

		// Calculate the delta in responses sent.
		if value >= previous.Value {
			// New value is larger, we assume we have contiguous data.
			interval.Responses = value - previous.Value
		} else {
			// We have either Kea restart, reset, or statistic rollover. This value
			// then represents the number packets sent since that event occurred.
			interval.Responses = value
		}

		err = dbmodel.AddRpsInterval(rpsWorker.db, interval)
	}

	// Always update the last reported values for the Daemon.
	rpsWorker.PreviousRps[daemonID] = StatSample{timestamp, value}

	return err
}

// Update the RPS value for both intervals for given daemon.
// Uses the RpsInterval table contents to get the total responses and duration
// for both intervals and then updates the Daemon's statistics in the db.
func (rpsWorker *RpsWorker) updateKeaDaemonRpsStats(daemon *dbmodel.Daemon) error {
	endTime := storkutil.UTCNow()
	startTime1 := endTime.Add(-rpsWorker.Interval1)
	daemonID := daemon.KeaDaemon.DaemonID

	// Fetch interval totals for interval 1.
	rps1, err := dbmodel.GetTotalRpsOverIntervalForDaemon(rpsWorker.db, startTime1, endTime, daemonID)
	if err != nil {
		return errors.WithMessagef(err, "query for RPS interval 1 data failed")
	}

	// Calculate RPS for interval 1.
	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1 = calculateRps(rps1)

	// Fetch interval totals for interval 1.
	startTime2 := endTime.Add(-rpsWorker.Interval2)
	rps2, err := dbmodel.GetTotalRpsOverIntervalForDaemon(rpsWorker.db, startTime2, endTime, daemonID)
	if err != nil {
		return errors.WithMessagef(err, "query for RPS interval 2 data failed")
	}

	// Calculate RPS for interval 2.
	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2 = calculateRps(rps2)

	// Update the daemon statistics.
	log.Printf("Updating KeaDHCPDaemonStats: %+v", daemon.KeaDaemon.KeaDHCPDaemon.Stats)
	return dbmodel.UpdateDaemon(rpsWorker.db, daemon)
}

// Calculate the RPS for the first row in a set of RpsIntervals.
func calculateRps(totals []*dbmodel.RpsInterval) float32 {
	if len(totals) == 0 {
		return 0
	}

	responses := totals[0].Responses
	duration := totals[0].Duration
	if responses <= 0 || duration <= 0 {
		return 0
	}

	// Return the rate.
	return float32(responses) / float32(duration)
}
