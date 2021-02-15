package agentcomm

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Structure representing a periodic puller which is configured to
// execute a function specified by a caller according to the timer
// interval specified in the database. The user's fuction typically
// pulls and manipulates the data from multiple apps.
type PeriodicPuller struct {
	pullerName          string
	intervalSettingName string
	pullFunc            func() (int, error)
	DB                  *dbops.PgDB
	Agents              ConnectedAgents
	Ticker              *time.Ticker
	Interval            int64
	Active              bool
	pauseCount          uint16
	Done                chan bool
	Wg                  *sync.WaitGroup
	mutex               *sync.Mutex
}

const InactiveInterval int64 = 60

// Creates an instance of a new periodic puller. The periodic puller offers a mechanism
// to periodically trigger an action. This action is supplied as a function instance.
// This function is executed within a goroutine periodically according to the timer
// interval available in the database. The intervalSettingName is a name of this
// setting in the database. The pullerName is used for logging purposes.
func NewPeriodicPuller(db *dbops.PgDB, agents ConnectedAgents, pullerName, intervalSettingName string, pullFunc func() (int, error)) (*PeriodicPuller, error) {
	log.Printf("starting %s Puller", pullerName)

	interval, err := dbmodel.GetSettingInt(db, intervalSettingName)
	if err != nil {
		return nil, err
	}

	// if interval in db is 0 then it means that puller is disabled,
	// but it needs to check from time to time the interval in db
	// to reenable itself when it gets to be > 0. When it is disabled
	// then it checks db every 60 seconds (InactiveInterval).
	active := true
	if interval <= 0 {
		interval = InactiveInterval
		active = false
	}

	periodicPuller := &PeriodicPuller{
		pullerName:          pullerName,
		intervalSettingName: intervalSettingName,
		pullFunc:            pullFunc,
		DB:                  db,
		Agents:              agents,
		Ticker:              time.NewTicker(time.Duration(interval) * time.Second),
		Interval:            interval,
		Active:              active,
		pauseCount:          0,
		Done:                make(chan bool),
		Wg:                  &sync.WaitGroup{},
		mutex:               &sync.Mutex{},
	}

	periodicPuller.Wg.Add(1)
	go periodicPuller.pullerLoop()

	log.Printf("started %s Puller", periodicPuller.pullerName)
	return periodicPuller, nil
}

// Terminates the puller, i.e. the puller no longer triggers the
// user defined function.
func (puller *PeriodicPuller) Shutdown() {
	log.Printf("stopping %s Puller", puller.pullerName)
	puller.Done <- true
	puller.Wg.Wait()
	log.Printf("stopped %s Puller", puller.pullerName)
}

// Temporarily stops the timer triggering the puller action. This function
// is called internally by the puller while running the puller action to
// avoid the situation that after long lasting action it is triggered again
// shortly. It can also be called externally if the puller action would
// be in conflict with some other operation during which the puller is
// paused. This function allows for being called multiple times and the
// timer is resumed after calling Unpause the same number of times. This
// is useful when the puller can be potentially paused and unpaused from
// different parts of the code concurrently.
func (puller *PeriodicPuller) Pause() {
	puller.mutex.Lock()
	defer puller.mutex.Unlock()
	puller.Ticker.Stop()
	puller.pauseCount++
}

// Checks if the puller is currently paused.
func (puller *PeriodicPuller) Paused() bool {
	puller.mutex.Lock()
	defer puller.mutex.Unlock()
	return puller.pauseCount > 0
}

// Unpause implementation which optionally locks the puller's mutex.
// This function is internally called by Unpause() and Reset(). Note
// that Reset() locks the mutex on its own so the lock argument is
// set to false in this case.
func (puller *PeriodicPuller) unpause(lock bool, interval ...int64) {
	intervals := interval
	if len(intervals) > 1 {
		// This should not happen.
		panic("Resume accepts one or zero interval values")
	}
	if lock {
		puller.mutex.Lock()
		defer puller.mutex.Unlock()
	}
	if puller.pauseCount > 0 {
		puller.pauseCount--
	}
	// Unpause() called for all earlier calls to Pause(), so we can resume
	// the puller action.
	if puller.pauseCount == 0 {
		if len(intervals) > 0 {
			// Override the interval.
			puller.Interval = intervals[0]
		}
		// Reschedule the timer.
		puller.Ticker.Reset(time.Duration(puller.Interval) * time.Second)
	}
}

// Unpauses the puller. The optional interval parameter may contain
// one interval value which overrides the current interval. If the interval
// is not specified, the current interval is used.
func (puller *PeriodicPuller) Unpause(interval ...int64) {
	puller.unpause(true, interval...)
}

// Reschedule the puller timer to a new interval. It forcibly stops
// the puller and reschedules to the new interval.
func (puller *PeriodicPuller) Reset(interval int64) {
	puller.mutex.Lock()
	defer puller.mutex.Unlock()
	puller.Ticker.Stop()
	puller.pauseCount = 0
	puller.unpause(false, interval)
}

// This function controls the timing of the function execution and captures the
// termination signal.
func (puller *PeriodicPuller) pullerLoop() {
	defer puller.Wg.Done()
	for {
		select {
		// every N seconds execute user defined function
		case <-puller.Ticker.C:
			if puller.Active {
				// Temporarily stop the puller while running the external action.
				// It will be resumed when the action ends.
				puller.Pause()
				_, err := puller.pullFunc()
				puller.Unpause()
				if err != nil {
					log.Errorf("errors were encountered while pulling data from apps: %+v", err)
				}
			}
		// wait for done signal from shutdown function
		case <-puller.Done:
			// Make sure this function is never called again.
			puller.Pause()
			return
		}

		// Check if the interval has changed in the database. If so, recreate the ticker.
		interval, err := dbmodel.GetSettingInt(puller.DB, puller.intervalSettingName)
		if err != nil {
			log.Errorf("problem with getting interval setting %s from db: %+v",
				puller.intervalSettingName, err)
		} else {
			puller.mutex.Lock()
			pullerInterval := puller.Interval
			puller.mutex.Unlock()
			if interval <= 0 && puller.Active {
				// if puller should be disabled but it is active then
				if pullerInterval != InactiveInterval {
					puller.Reset(InactiveInterval)
				}
				puller.Active = false
			} else if interval > 0 && interval != pullerInterval {
				// if puller interval is changed and is not 0 (disabled)
				puller.Reset(interval)
				puller.Active = true
			}
		}
	}
}
