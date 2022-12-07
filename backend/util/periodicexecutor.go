package storkutil

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Structure representing a periodic executor which is configured to
// execute a function specified by a caller according to the timer
// interval specified.
type PeriodicExecutor struct {
	name            string
	executorFunc    func() error
	interval        int64
	ticker          *time.Ticker
	active          bool
	pauseCount      uint16
	done            chan bool
	wg              *sync.WaitGroup
	mutex           *sync.Mutex
	getIntervalFunc func() (int64, error)
}

// Interval is used while the puller is inactive to check if it was re-enabled.
const InactiveInterval int64 = 60

// Creates an instance of a new periodic executor. The periodic executor offers a mechanism
// to periodically trigger an action. This action is supplied as a function instance.
// This function is executed within a goroutine periodically according to the timer
// interval calculated by `getIntervalFunc`. It accepts previous interval and returns next value.
func NewPeriodicExecutor(name string, executorFunc func() error, getIntervalFunc func() (int64, error)) (*PeriodicExecutor, error) {
	log.Printf("Starting %s", name)

	interval, err := getIntervalFunc()
	if err != nil {
		return nil, err
	}

	// if interval is 0 then it means that executor is disabled,
	// but it needs to check from time to time the interval using inactiveFunc
	// to reenable itself when it gets to be > 0. When it is disabled
	// then it checks db every 60 seconds (InactiveInterval).
	active := true
	if interval <= 0 {
		interval = InactiveInterval
		active = false
	}

	periodicExecutor := &PeriodicExecutor{
		name:            name,
		executorFunc:    executorFunc,
		ticker:          time.NewTicker(time.Duration(interval) * time.Second),
		active:          active,
		pauseCount:      0,
		done:            make(chan bool),
		wg:              &sync.WaitGroup{},
		mutex:           &sync.Mutex{},
		interval:        interval,
		getIntervalFunc: getIntervalFunc,
	}

	periodicExecutor.wg.Add(1)
	go periodicExecutor.executorLoop()

	log.Printf("Started %s", periodicExecutor.name)
	return periodicExecutor, nil
}

// Terminates the executor, i.e. the executor no longer triggers the
// user defined function.
func (executor *PeriodicExecutor) Shutdown() {
	log.Printf("Stopping %s", executor.name)
	executor.done <- true
	executor.wg.Wait()
	log.Printf("Stopped %s", executor.name)
}

// Temporarily stops the timer triggering the executor action. This function
// is called internally by the executor while running the executor action to
// avoid the situation that after long lasting action it is triggered again
// shortly. It can also be called externally if the executor action would
// be in conflict with some other operation during which the executor is
// paused. This function allows for being called multiple times and the
// timer is resumed after calling Unpause the same number of times. This
// is useful when the executor can be potentially paused and unpaused from
// different parts of the code concurrently.
func (executor *PeriodicExecutor) Pause() {
	executor.mutex.Lock()
	defer executor.mutex.Unlock()
	executor.ticker.Stop()
	executor.pauseCount++
}

// Checks if the executor is currently paused.
func (executor *PeriodicExecutor) Paused() bool {
	executor.mutex.Lock()
	defer executor.mutex.Unlock()
	return executor.pauseCount > 0
}

// Unpause implementation which optionally locks the executor's mutex.
// This function is internally called by Unpause() and Reset(). Note
// that Reset() locks the mutex on its own so the lock argument is
// set to false in this case.
func (executor *PeriodicExecutor) unpause(lock bool, intervals ...int64) {
	if len(intervals) > 1 {
		// This should not happen.
		panic("Resume accepts one or zero interval values")
	}
	if lock {
		executor.mutex.Lock()
		defer executor.mutex.Unlock()
	}
	if executor.pauseCount > 0 {
		executor.pauseCount--
	}
	// Unpause() called for all earlier calls to Pause(), so we can resume
	// the executor action.
	if executor.pauseCount == 0 {
		if len(intervals) > 0 {
			// Override the interval.
			executor.interval = intervals[0]
		}
		// Reschedule the timer.
		executor.ticker.Reset(time.Duration(executor.interval) * time.Second)
	}
}

// Unpauses the executor. The optional interval parameter may contain
// one interval value which overrides the current interval. If the interval
// is not specified, the current interval is used.
func (executor *PeriodicExecutor) Unpause(interval ...int64) {
	executor.unpause(true, interval...)
}

// Return the current interval in seconds.
func (executor *PeriodicExecutor) GetInterval() int64 {
	executor.mutex.Lock()
	defer executor.mutex.Unlock()
	return executor.interval
}

// Reschedule the executor timer to a new interval. It forcibly stops
// the executor and reschedules to the new interval.
func (executor *PeriodicExecutor) Reset(interval int64) {
	executor.mutex.Lock()
	defer executor.mutex.Unlock()
	executor.ticker.Stop()
	executor.pauseCount = 0
	executor.unpause(false, interval)
}

// This function controls the timing of the function execution and captures the
// termination signal.
func (executor *PeriodicExecutor) executorLoop() {
	defer executor.wg.Done()
	for {
		select {
		// every N seconds execute user defined function
		case <-executor.ticker.C:
			if executor.active {
				// Temporarily stop the executor while running the external action.
				// It will be resumed when the action ends.
				executor.Pause()
				err := executor.executorFunc()
				executor.Unpause()
				if err != nil {
					log.Errorf("Errors were encountered while pulling data from apps: %+v", err)
				}
			}
		// wait for done signal from shutdown function
		case <-executor.done:
			// Make sure this function is never called again.
			executor.Pause()
			return
		}

		// Check if the interval has changed. If so, recreate the ticker.
		interval, err := executor.getIntervalFunc()
		if err != nil {
			log.Errorf("Problem getting interval: %+v", err)
			return
		}

		executor.mutex.Lock()
		executorInterval := executor.interval
		executor.mutex.Unlock()

		if interval <= 0 && executor.active {
			// if executor should be disabled but it is active then
			if executorInterval != InactiveInterval {
				executor.Reset(InactiveInterval)
			}
			executor.active = false
		} else if interval > 0 && interval != executorInterval {
			// if executor interval is changed and is not 0 (disabled)
			executor.Reset(interval)
			executor.active = true
		}
	}
}

// Returns the executor name.
func (executor *PeriodicExecutor) GetName() string {
	return executor.name
}
