package agentcomm

import (
	"io"
	"reflect"
	"sync"

	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

var _ io.Closer = (*CommStatsWrapper)(nil)

// Enumeration representing a type of the communication state transition
// while the server tries to send a gRPC command to an agent.
type CommErrorTransition int

const (
	// No communication issue previously and now.
	CommErrorNone CommErrorTransition = iota
	// New communication issue after successful communication earlier.
	CommErrorNew
	// Communication issue is gone after unsuccessful communication earlier.
	CommErrorReset
	// Communication issue persists.
	CommErrorContinued
)

// Holds runtime statistics of the communication with a given agent and
// with the daemons behind this agent. The statistics are maintained by the
// logic in the agentcomm package but can be read from other packages.
// In particular, the REST API handlers read the statistics to present
// communication issues in the UI. The data held in this structure are
// prone to the race conditions. The data pullers run in the goroutines
// and call the gRPC handlers in this package. These calls can cause
// races with the calls made from the REST API handlers. Thus, it is
// important to use the mutex instance held in this structure to lock the
// statistics for writing and reading. The caller is responsible for
// this locking. The functions implemented here do not lock because
// locking outside is more efficient when several operations on the
// statistics must be performed.
type CommStats struct {
	// Number of the communication errors with a Stork Agent for a
	// particular type of gRPC message.
	agentCommErrors map[string]int64
	// Number of the communication errors with a specific access point of a
	// particular kind of daemon.
	daemonCommErrors map[daemonname.Name]map[dbmodel.AccessPointType]int64
	// A mutex to be used when the data is accessed or updated. This
	// mutex is returned to the caller and the caller is responsible
	// for locking and unlocking the mutex.
	mutex *sync.RWMutex
}

// Instantiates communication statistics for a new agent.
func NewAgentStats() *CommStats {
	return &CommStats{
		agentCommErrors:  make(map[string]int64),
		daemonCommErrors: make(map[daemonname.Name]map[dbmodel.AccessPointType]int64),
		mutex:            &sync.RWMutex{},
	}
}

// Communication errors with an agent are grouped by the type of the
// gRPC command. The gRPC request types are converted to strings by
// this function and used as a key in the AgentCommStats.agentCommErrors
// map.
func encodeAsCommStatsRequestName(request any) string {
	requestType := reflect.TypeOf(request)
	if requestType.Kind() == reflect.Pointer {
		return requestType.Elem().Name()
	}
	return requestType.Name()
}

// Increases an error count for a selected daemon type by 1.
// Returns an updated count.
func (stats *CommStats) increaseDaemonErrorCount(daemonName daemonname.Name, accessPointType dbmodel.AccessPointType) int64 {
	return stats.increaseDaemonErrorCountBy(daemonName, accessPointType, 1)
}

// Increases an error count for a selected daemon type by
// an arbitrary number. Returns an updated count.
func (stats *CommStats) increaseDaemonErrorCountBy(daemonName daemonname.Name, accessPointType dbmodel.AccessPointType, increment int64) int64 {
	if _, ok := stats.daemonCommErrors[daemonName]; !ok {
		stats.daemonCommErrors[daemonName] = make(map[dbmodel.AccessPointType]int64)
	}
	if _, ok := stats.daemonCommErrors[daemonName][accessPointType]; !ok {
		stats.daemonCommErrors[daemonName][accessPointType] = 0
	}
	stats.daemonCommErrors[daemonName][accessPointType] += increment
	return stats.daemonCommErrors[daemonName][accessPointType]
}

// Updates an error count for a daemon using a specified new count.
// The new counter of 0 indicates that any previous communication
// issues with the daemon are gone and causes the current error
// count to be reset. A positive error count indicates an ongoing
// communication issue and increases the number of unsuccessful
// communication attempts. The returned transition state allows the
// caller for determining if there is a new communication problem or
// the previous no gone. Based on that, the caller can issue
// appropriate events.
func (stats *CommStats) updateDaemonErrorCount(daemonName daemonname.Name, accessPointType dbmodel.AccessPointType, newCount int64) CommErrorTransition {
	currentCount := stats.getDaemonErrorCount(daemonName, accessPointType)
	switch {
	case newCount == 0 && currentCount == 0:
		return CommErrorNone
	case newCount > 0 && currentCount > 0:
		_ = stats.increaseDaemonErrorCountBy(daemonName, accessPointType, newCount)
		return CommErrorContinued
	case newCount == 0 && currentCount > 0:
		stats.resetDaemonErrorCount(daemonName, accessPointType)
		return CommErrorReset
	case newCount > 0 && currentCount == 0:
		_ = stats.increaseDaemonErrorCountBy(daemonName, accessPointType, newCount)
		return CommErrorNew
	default:
		return CommErrorNone
	}
}

// Increases communication error count with an agent by 1. Returns
// an updated count.
func (stats *CommStats) IncreaseAgentErrorCount(request any) int64 {
	requestName := encodeAsCommStatsRequestName(request)
	stats.agentCommErrors[requestName]++
	return stats.agentCommErrors[requestName]
}

// Resets the communication error count with a daemon to 0.
func (stats *CommStats) resetDaemonErrorCount(daemonName daemonname.Name, accessPointType dbmodel.AccessPointType) {
	if _, ok := stats.daemonCommErrors[daemonName]; !ok {
		return
	}
	if _, ok := stats.daemonCommErrors[daemonName][accessPointType]; !ok {
		return
	}
	delete(stats.daemonCommErrors[daemonName], accessPointType)
}

// Resets the communication error count with an agent to 0.
func (stats *CommStats) ResetAgentErrorCount(request any) {
	delete(stats.agentCommErrors, encodeAsCommStatsRequestName(request))
}

// Returns a current communication error count with a daemon.
func (stats *CommStats) getDaemonErrorCount(daemonName daemonname.Name, accessPointType dbmodel.AccessPointType) int64 {
	if daemonCommErrors, ok := stats.daemonCommErrors[daemonName]; ok {
		return daemonCommErrors[accessPointType]
	}
	return 0
}

// Returns a current communication error count with an agent.
func (stats *CommStats) GetAgentErrorCount(request any) int64 {
	if agentCommErrors, ok := stats.agentCommErrors[encodeAsCommStatsRequestName(request)]; ok {
		return agentCommErrors
	}
	return 0
}

// Returns a communication error count with an agent for all gRPC request types.
func (stats *CommStats) GetTotalAgentErrorCount() int64 {
	var totalErrorCount int64
	for _, errorCount := range stats.agentCommErrors {
		totalErrorCount += errorCount
	}
	return totalErrorCount
}

// Returns an object implementing a Kea-specific interface for updating the statistics.
func (stats *CommStats) GetKeaStats() *CommStatsKea {
	return &CommStatsKea{
		comm: stats,
	}
}

// Returns an object implementing a BIND 9-specific interface for updating the
// statistics.
func (stats *CommStats) GetBind9Stats() *CommStatsBind9 {
	return &CommStatsBind9{
		comm: stats,
	}
}

// Provides a Kea-specific way of updating the error count for given daemons.
type CommStatsKea struct {
	comm *CommStats
}

// Updates an error count for a daemon using a specified new count.
// The new counter of 0 indicates that any previous communication
// issues with the daemon are gone and causes the current error
// count to be reset. A positive error count indicates an ongoing
// communication issue and increases the number of unsuccessful
// communication attempts. The returned transition state allows the
// caller for determining if there is a new communication problem or
// the previous no gone. Based on that, the caller can issue
// appropriate events.
func (ks *CommStatsKea) UpdateErrorCount(daemonName daemonname.Name, count int64) CommErrorTransition {
	return ks.comm.updateDaemonErrorCount(daemonName, dbmodel.AccessPointControl, count)
}

// Returns a number of communication errors with a Kea daemon for a
// particular channel.
func (ks *CommStatsKea) GetErrorCount(daemonName daemonname.Name) int64 {
	return ks.comm.getDaemonErrorCount(daemonName, dbmodel.AccessPointControl)
}

// Increases communication error count with a Kea daemon by 1. Returns
// an updated count.
func (ks *CommStatsKea) IncreaseErrorCount(daemonName daemonname.Name) int64 {
	return ks.comm.increaseDaemonErrorCount(daemonName, dbmodel.AccessPointControl)
}

// Increases communication error count with a Kea daemon by a provided value.
// Returns an updated count.
func (ks *CommStatsKea) IncreaseErrorCountBy(daemonName daemonname.Name, value int64) int64 {
	return ks.comm.increaseDaemonErrorCountBy(daemonName, dbmodel.AccessPointControl, value)
}

// Provides a BIND 9-specific way of updating the error count for given daemons.
type CommStatsBind9 struct {
	comm *CommStats
}

// Returns a number of communication errors with a BIND 9 daemon for a
// particular channel.
func (bs *CommStatsBind9) GetErrorCount(accessPointType dbmodel.AccessPointType) int64 {
	return bs.comm.getDaemonErrorCount(daemonname.Bind9, accessPointType)
}

// Increases communication error count with a BIND 9 daemon by 1. Returns
// an updated count.
func (bs *CommStatsBind9) IncreaseErrorCount(accessPointType dbmodel.AccessPointType) int64 {
	return bs.comm.increaseDaemonErrorCount(daemonname.Bind9, accessPointType)
}

// Increases communication error count with a BIND 9 daemon by a provided value.
// Returns an updated count.
func (bs *CommStatsBind9) IncreaseErrorCountBy(accessPointType dbmodel.AccessPointType, value int64) int64 {
	return bs.comm.increaseDaemonErrorCountBy(daemonname.Bind9, accessPointType, value)
}

// Resets the communication error count with a BIND 9 daemon to 0.
func (bs *CommStatsBind9) ResetErrorCount(accessPointType dbmodel.AccessPointType) {
	bs.comm.resetDaemonErrorCount(daemonname.Bind9, accessPointType)
}

// A wrapper for AgentCommStats which locks the stats for reading and
// implements the Closer interface releasing the lock when desired.
type CommStatsWrapper struct {
	/// Agent communication stats to be made available in a safe manner.
	agentCommStats *CommStats
}

// Instantiates the stats wrapper and locks the stats for reading.
func NewCommStatsWrapper(stats *CommStats) *CommStatsWrapper {
	stats.mutex.RLock()
	return &CommStatsWrapper{
		agentCommStats: stats,
	}
}

// Returns the wrapped agent communication stats. The returned stats are locked
// for reading. An attempt to write to the stats can cause a race condition.
func (wrapper *CommStatsWrapper) GetStats() *CommStats {
	return wrapper.agentCommStats
}

// Releases the lock.
func (wrapper *CommStatsWrapper) Close() error {
	wrapper.agentCommStats.mutex.RUnlock()
	return nil
}
