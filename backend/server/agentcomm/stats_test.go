package agentcomm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

type requestTypeFoo struct{}

type requestTypeBar struct{}

// Test instantiating a structure holding communication errors.
func TestNewCommStats(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	require.NotNil(t, stats.agentCommErrors)
	require.NotNil(t, stats.daemonCommErrors)
}

// Test increasing an error count for a selected Kea daemon type by 1.
func TestIncreaseKeaErrorCount(t *testing.T) {
	stats := NewAgentStats()
	keaStats := stats.GetKeaStats()
	require.NotNil(t, keaStats)
	require.Zero(t, keaStats.GetErrorCount(daemonname.CA))
	require.Zero(t, keaStats.GetErrorCount(daemonname.DHCPv4))
	keaStats.IncreaseErrorCount(daemonname.CA)
	require.EqualValues(t, 1, keaStats.GetErrorCount(daemonname.CA))
	require.Zero(t, keaStats.GetErrorCount(daemonname.DHCPv4))
	keaStats.IncreaseErrorCount(daemonname.CA)
	require.EqualValues(t, 2, keaStats.GetErrorCount(daemonname.CA))
	require.Zero(t, keaStats.GetErrorCount(daemonname.DHCPv4))
	keaStats.IncreaseErrorCount(daemonname.DHCPv4)
	require.EqualValues(t, 2, keaStats.GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, keaStats.GetErrorCount(daemonname.DHCPv4))
}

// Test increasing an error count for a selected Kea daemon type by any value.
func TestIncreaseByKeaErrorCount(t *testing.T) {
	// Arrange
	stats := NewAgentStats()
	keaStats := stats.GetKeaStats()

	// Act
	keaStats.IncreaseErrorCountBy(daemonname.CA, 5)
	keaStats.IncreaseErrorCountBy(daemonname.DHCPv4, 3)
	keaStats.IncreaseErrorCountBy(daemonname.CA, 2)
	keaStats.IncreaseErrorCountBy(daemonname.DHCPv4, -2)

	// Assert
	require.EqualValues(t, 7, keaStats.GetErrorCount(daemonname.CA))
	require.EqualValues(t, 1, keaStats.GetErrorCount(daemonname.DHCPv4))
}

// Test updating an error count for a daemon and tracking the communication
// error transition based on the updated count.
func TestUpdateKeaErrorCount(t *testing.T) {
	stats := NewAgentStats()
	keaStats := stats.GetKeaStats()
	require.NotNil(t, keaStats)
	require.Equal(t, CommErrorNone, keaStats.UpdateErrorCount(daemonname.CA, 0))
	require.Equal(t, CommErrorNew, keaStats.UpdateErrorCount(daemonname.CA, 2))
	require.Equal(t, CommErrorContinued, keaStats.UpdateErrorCount(daemonname.CA, 3))
	require.Equal(t, CommErrorReset, keaStats.UpdateErrorCount(daemonname.CA, 0))
}

// Test increasing an error count for a selected BIND 9 channel type by 1.
func TestIncreaseBind9ErrorCount(t *testing.T) {
	stats := NewAgentStats()
	bind9Stats := stats.GetBind9Stats()
	require.NotNil(t, bind9Stats)
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
	require.EqualValues(t, 1, bind9Stats.IncreaseErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 1, bind9Stats.GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
	require.EqualValues(t, 2, bind9Stats.IncreaseErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 2, bind9Stats.GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
	require.EqualValues(t, 1, bind9Stats.IncreaseErrorCount(dbmodel.AccessPointStatistics))
	require.EqualValues(t, 2, bind9Stats.GetErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 1, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test increasing an error count for a selected BIND 9 channel type by any value.
func TestIncreaseByBind9ErrorCount(t *testing.T) {
	// Arrange
	stats := NewAgentStats()
	bind9Stats := stats.GetBind9Stats()

	// Act
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointControl, 4)
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointStatistics, 6)
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointControl, 3)
	bind9Stats.IncreaseErrorCountBy(dbmodel.AccessPointStatistics, -2)

	// Assert
	require.EqualValues(t, 7, bind9Stats.GetErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 4, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test resetting the communication error count with a BIND 9 daemon to 0.
func TestResetBind9ErrorCount(t *testing.T) {
	stats := NewAgentStats()
	bind9Stats := stats.GetBind9Stats()
	bind9Stats.ResetErrorCount(dbmodel.AccessPointStatistics)
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
	bind9Stats.IncreaseErrorCount(dbmodel.AccessPointStatistics)
	bind9Stats.IncreaseErrorCount(dbmodel.AccessPointStatistics)
	require.EqualValues(t, 2, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
	bind9Stats.ResetErrorCount(dbmodel.AccessPointStatistics)
	require.Zero(t, bind9Stats.GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test instantiating communication statistics for a new agent.
func TestNewAgentStats(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	require.NotNil(t, stats.agentCommErrors)
	require.NotNil(t, stats.daemonCommErrors)
	require.NotNil(t, stats.mutex)
}

// Test increasing communication error count with an agent by 1.
func TestIncreaseAgentErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	require.EqualValues(t, 1, stats.IncreaseAgentErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 1, stats.GetAgentErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetAgentErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 2, stats.IncreaseAgentErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 2, stats.GetAgentErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetAgentErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 3, stats.IncreaseAgentErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 3, stats.GetAgentErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetAgentErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 1, stats.IncreaseAgentErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 3, stats.GetAgentErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 1, stats.GetAgentErrorCount(&requestTypeBar{}))
}

// Test resetting the communication error count with an agent to 0.
func TestResetAgentErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	for i := 0; i < 10; i++ {
		require.EqualValues(t, i+1, stats.IncreaseAgentErrorCount(requestTypeFoo{}))
	}
	require.EqualValues(t, 10, stats.GetAgentErrorCount(requestTypeFoo{}))
	require.Zero(t, stats.GetAgentErrorCount(requestTypeBar{}))

	stats.ResetAgentErrorCount(requestTypeFoo{})
	require.Zero(t, stats.GetAgentErrorCount(requestTypeFoo{}))
	require.Zero(t, stats.GetAgentErrorCount(requestTypeBar{}))
}

// Test returning a communication error count with an agent for all
// gRPC request types.
func TestGetTotalErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	for i := 0; i < 10; i++ {
		require.EqualValues(t, i+1, stats.IncreaseAgentErrorCount(requestTypeFoo{}))
	}
	for i := 0; i < 30; i++ {
		require.EqualValues(t, i+1, stats.IncreaseAgentErrorCount(requestTypeBar{}))
	}
	require.EqualValues(t, 10, stats.GetAgentErrorCount(requestTypeFoo{}))
	require.EqualValues(t, 30, stats.GetAgentErrorCount(requestTypeBar{}))
	require.EqualValues(t, 40, stats.GetTotalAgentErrorCount())
}

// Test returning Kea communication error stats for a selected daemon by ID.
func TestGetKeaCommErrorStats(t *testing.T) {
	stats := NewAgentStats()
	keaStats := stats.GetKeaStats()
	require.NotNil(t, keaStats)
	require.EqualValues(t, 1, keaStats.IncreaseErrorCount(daemonname.CA))
	require.EqualValues(t, 1, stats.GetKeaStats().GetErrorCount(daemonname.CA))
	require.Zero(t, stats.GetKeaStats().GetErrorCount(daemonname.DHCPv4))
}

// Test returning BIND 9 communication error stats for a selected daemon by ID.
func TestGetBind9CommErrorStats(t *testing.T) {
	stats := NewAgentStats()
	bind9Stats := stats.GetBind9Stats()
	require.NotNil(t, bind9Stats)
	require.EqualValues(t, 1, bind9Stats.IncreaseErrorCount(dbmodel.AccessPointControl))
	require.EqualValues(t, 1, stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointControl))
	require.Zero(t, stats.GetBind9Stats().GetErrorCount(dbmodel.AccessPointStatistics))
}

// Test that returned statistics is locked and can be unlocked with the
// call to the Close function.
func TestAgentCommStatsWrapperLock(t *testing.T) {
	stats := NewAgentStats()
	wrapper := NewCommStatsWrapper(stats)
	require.NotNil(t, wrapper)
	mutex := wrapper.GetStats().mutex
	// The mutex should be locked for updates. This attempt should
	// fail and return false.
	require.False(t, mutex.TryLock())
	// Call Close to release the mutex lock. Protect against panics because
	// double release could cause it.
	require.NotPanics(t, func() { wrapper.Close() })
	// This time locking should succeed.
	require.True(t, mutex.TryLock())
	defer mutex.Unlock()
}

// Test extracting statistics from the returned statistics wrapper.
func TestAgentCommStatsWrapperGetStats(t *testing.T) {
	stats := NewAgentStats()
	wrapper := NewCommStatsWrapper(stats)
	require.NotNil(t, wrapper)
	defer wrapper.Close()
	returnedStats := wrapper.GetStats()
	require.Equal(t, stats, returnedStats)
}
