package agentcomm

import (
	"io"
	"reflect"
	"sync"
)

var _ io.Closer = (*AgentCommStatsWrapper)(nil)

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

// Enumeration representing a Kea daemon type for which one of the
// statistics manipulation functions is called.
type KeaDaemonType int

const (
	KeaDaemonDHCPv4 KeaDaemonType = iota
	KeaDaemonDHCPv6
	KeaDaemonD2
	KeaDaemonCA
	KeaDaemonUnknown
)

// Enumeration representing a communication channel with a BIND 9
// daemon (i.e., RNDC or statistics channel).
type Bind9ChannelType int

const (
	Bind9ChannelRNDC Bind9ChannelType = iota
	Bind9ChannelStats
)

// Holds runtime communication statistics with Kea daemons on single machine.
type KeaAppCommErrorStats struct {
	// Communication error counts for different Kea daemons.
	errorCounts map[KeaDaemonType]int64
}

// Holds runtime communication statistics with BIND 9 daemon on a single machine.
type Bind9AppCommErrorStats struct {
	// Communication error counts for different BIND 9
	// communication channels.
	errorCounts map[Bind9ChannelType]int64
}

// A wrapper for AgentCommStats which locks the stats for reading and
// implements the Closer interface releasing the lock when desired.
type AgentCommStatsWrapper struct {
	/// Agent communication stats to be made available in a safe manner.
	agentCommStats *AgentCommStats
}

// Holds runtime statistics of the communication with a given agent and
// with the apps behind this agent. The statistics are maintained by the
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
type AgentCommStats struct {
	// Number of the communication errors with a Stork Agent for a
	// particular type of gRPC message.
	agentCommErrors map[string]int64
	// Number of the communication errors with a Kea instance having
	// a particular App ID.
	keaCommErrors map[int64]*KeaAppCommErrorStats
	// Number of the communication errors with a BIND 9 instance having
	// a particular App ID.
	bind9CommErrors map[int64]*Bind9AppCommErrorStats
	// A mutex to be used when the data is accessed or updated. This
	// mutex is returned to the caller and the caller is responsible
	// for locking and unlocking the mutex.
	mutex *sync.RWMutex
}

// Instantiates the stats wrapper and locks the stats for reading.
func NewAgentCommStatsWrapper(stats *AgentCommStats) *AgentCommStatsWrapper {
	stats.mutex.RLock()
	return &AgentCommStatsWrapper{
		agentCommStats: stats,
	}
}

// Returns the wrapped agent communication stats. The returned stats are locked
// for reading. An attempt to write to the stats can cause a race condition.
func (wrapper *AgentCommStatsWrapper) GetStats() *AgentCommStats {
	return wrapper.agentCommStats
}

// Releases the lock.
func (wrapper *AgentCommStatsWrapper) Close() error {
	wrapper.agentCommStats.mutex.RUnlock()
	return nil
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

// Convenience function converting a Kea daemon name to its type.
func GetKeaDaemonTypeFromName(daemonName string) KeaDaemonType {
	switch daemonName {
	case "dhcp4":
		return KeaDaemonDHCPv4
	case "dhcp6":
		return KeaDaemonDHCPv6
	case "d2":
		return KeaDaemonD2
	case "", "ca":
		return KeaDaemonCA
	default:
		return KeaDaemonUnknown
	}
}

// Instantiates the communication statistics for a new Kea instance.
func NewKeaAppCommErrorStats() *KeaAppCommErrorStats {
	return &KeaAppCommErrorStats{
		errorCounts: make(map[KeaDaemonType]int64),
	}
}

// Increases an error count for a selected Kea daemon type by 1.
// Returns an updated count.
func (stats *KeaAppCommErrorStats) IncreaseErrorCount(daemonType KeaDaemonType) int64 {
	stats.errorCounts[daemonType]++
	return stats.errorCounts[daemonType]
}

// Increases an error count for a selected Kea daemon type by
// an arbitrary number. Returns an updated count.
func (stats *KeaAppCommErrorStats) IncreaseErrorCountBy(daemonType KeaDaemonType, increment int64) int64 {
	stats.errorCounts[daemonType] += increment
	return stats.errorCounts[daemonType]
}

// Updates an error count for a daemon using a specified new count.
// The new counter of 0 indicates that any previous communication
// issues with the daemon are gobe and causes the current error
// count to be reset. A positive error count indicates an ongoing
// communication issue and increases the number of unsuccessful
// communication attempts. The returned transition state allows the
// caller for determining if there is a new communication problem or
// the previous no gone. Based on that, the caller can issue
// appropriate events.
func (stats *KeaAppCommErrorStats) UpdateErrorCount(daemonType KeaDaemonType, newCount int64) CommErrorTransition {
	currentCount := stats.GetErrorCount(daemonType)
	switch {
	case newCount == 0 && currentCount == 0:
		return CommErrorNone
	case newCount > 0 && currentCount > 0:
		_ = stats.IncreaseErrorCountBy(daemonType, newCount)
		return CommErrorContinued
	case newCount == 0 && currentCount > 0:
		stats.ResetErrorCount(daemonType)
		return CommErrorReset
	case newCount > 0 && currentCount == 0:
		_ = stats.IncreaseErrorCountBy(daemonType, newCount)
		return CommErrorNew
	default:
		return CommErrorNone
	}
}

// Resets an error count for a selected daemon to 0.
func (stats *KeaAppCommErrorStats) ResetErrorCount(daemonType KeaDaemonType) {
	delete(stats.errorCounts, daemonType)
}

// Returns a current error count for a selected daemon.
func (stats *KeaAppCommErrorStats) GetErrorCount(daemonType KeaDaemonType) int64 {
	if errorCount, ok := stats.errorCounts[daemonType]; ok {
		return errorCount
	}
	return 0
}

// Instantiates the communication statistics for a new BIND 9 instance.
func NewBind9AppCommErrorStats() *Bind9AppCommErrorStats {
	return &Bind9AppCommErrorStats{
		errorCounts: make(map[Bind9ChannelType]int64),
	}
}

// Increases an error count for a selected channel type by 1.
// Returns an updated count.
func (stats *Bind9AppCommErrorStats) IncreaseErrorCount(channelType Bind9ChannelType) int64 {
	stats.errorCounts[channelType]++
	return stats.errorCounts[channelType]
}

// Increases an error count for a selected channel type by an
// arbitrary number. Returns an updated count.
func (stats *Bind9AppCommErrorStats) IncreaseErrorCountBy(channelType Bind9ChannelType, increment int64) int64 {
	stats.errorCounts[channelType] += increment
	return stats.errorCounts[channelType]
}

// Resets an error count for a selected channel type to 0.
func (stats *Bind9AppCommErrorStats) ResetErrorCount(channelType Bind9ChannelType) {
	delete(stats.errorCounts, channelType)
}

// Returns a current error count for a selected channel type.
func (stats *Bind9AppCommErrorStats) GetErrorCount(channelType Bind9ChannelType) int64 {
	if errorCount, ok := stats.errorCounts[channelType]; ok {
		return errorCount
	}
	return 0
}

// Instantiates communication statistics for a new agent.
func NewAgentStats() *AgentCommStats {
	return &AgentCommStats{
		agentCommErrors: make(map[string]int64),
		keaCommErrors:   make(map[int64]*KeaAppCommErrorStats),
		bind9CommErrors: make(map[int64]*Bind9AppCommErrorStats),
		mutex:           &sync.RWMutex{},
	}
}

// Increases communication error count with an agent by 1. Returns
// an updated count.
func (stats *AgentCommStats) IncreaseErrorCount(request any) int64 {
	requestName := encodeAsCommStatsRequestName(request)
	stats.agentCommErrors[requestName]++
	return stats.agentCommErrors[requestName]
}

// Resets the communication error count with an agent to 0.
func (stats *AgentCommStats) ResetErrorCount(request any) {
	delete(stats.agentCommErrors, encodeAsCommStatsRequestName(request))
}

// Returns a current communication error count with an agent.
func (stats *AgentCommStats) GetErrorCount(request any) int64 {
	if agentCommErrors, ok := stats.agentCommErrors[encodeAsCommStatsRequestName(request)]; ok {
		return agentCommErrors
	}
	return 0
}

// Returns a communication error count with an agent for all
// gRPC request types.
func (stats *AgentCommStats) GetTotalErrorCount() int64 {
	var totalErrorCount int64
	for _, errorCount := range stats.agentCommErrors {
		totalErrorCount += errorCount
	}
	return totalErrorCount
}

// Returns Kea communication error stats for a selected app by ID.
// The returned pointer is guaranteed to be valid. If it doesn't
// initially exist it is created.
func (stats *AgentCommStats) GetKeaCommErrorStats(appID int64) *KeaAppCommErrorStats {
	if keaErrorStats, ok := stats.keaCommErrors[appID]; ok {
		return keaErrorStats
	}
	keaErrorStats := NewKeaAppCommErrorStats()
	stats.keaCommErrors[appID] = keaErrorStats
	return keaErrorStats
}

// Returns BIND 9 communication error stats for a selected app by ID.
// The returned pointer is guaranteed to be valid. If it doesn't
// initially exist it is created.
func (stats *AgentCommStats) GetBind9CommErrorStats(appID int64) *Bind9AppCommErrorStats {
	if bind9ErrorStats, ok := stats.bind9CommErrors[appID]; ok {
		return bind9ErrorStats
	}
	bind9ErrorStats := NewBind9AppCommErrorStats()
	stats.bind9CommErrors[appID] = bind9ErrorStats
	return bind9ErrorStats
}
