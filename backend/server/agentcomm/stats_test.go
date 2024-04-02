package agentcomm

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type requestTypeFoo struct{}

type requestTypeBar struct{}

// Test conversion of the daemon names to daemon types.
func TestKeaDaemonTypeFromName(t *testing.T) {
	require.Equal(t, KeaDaemonDHCPv4, GetKeaDaemonTypeFromName("dhcp4"))
	require.Equal(t, KeaDaemonDHCPv6, GetKeaDaemonTypeFromName("dhcp6"))
	require.Equal(t, KeaDaemonD2, GetKeaDaemonTypeFromName("d2"))
	require.Equal(t, KeaDaemonCA, GetKeaDaemonTypeFromName(""))
	require.Equal(t, KeaDaemonCA, GetKeaDaemonTypeFromName("ca"))
	require.Equal(t, KeaDaemonUnknown, GetKeaDaemonTypeFromName("foo"))
}

// Test instantiating a structure holding Kea communication errors.
func TestNewKeaAppCommErrorStats(t *testing.T) {
	stats := NewKeaAppCommErrorStats()
	require.NotNil(t, stats)
	require.NotNil(t, stats.errorCounts)
}

// Test increasing an error count for a selected Kea daemon type by 1.
func TestIncreaseKeaErrorCount(t *testing.T) {
	stats := NewKeaAppCommErrorStats()
	require.NotNil(t, stats)
	require.Zero(t, stats.GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetErrorCount(KeaDaemonDHCPv4))
	stats.IncreaseErrorCount(KeaDaemonCA)
	require.EqualValues(t, 1, stats.GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetErrorCount(KeaDaemonDHCPv4))
	stats.IncreaseErrorCount(KeaDaemonCA)
	require.EqualValues(t, 2, stats.GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetErrorCount(KeaDaemonDHCPv4))
	stats.IncreaseErrorCount(KeaDaemonDHCPv4)
	require.EqualValues(t, 2, stats.GetErrorCount(KeaDaemonCA))
	require.EqualValues(t, 1, stats.GetErrorCount(KeaDaemonDHCPv4))
}

// Test increasing an error count for a selected Kea daemon type by
// an arbitrary number.
func TestIncreaseKeaErrorCountBy(t *testing.T) {
	stats := NewKeaAppCommErrorStats()
	require.NotNil(t, stats)
	require.EqualValues(t, 5, stats.IncreaseErrorCountBy(KeaDaemonCA, 5))
	require.EqualValues(t, 5, stats.GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetErrorCount(KeaDaemonDHCPv4))
	require.EqualValues(t, 11, stats.IncreaseErrorCountBy(KeaDaemonCA, 6))
	require.EqualValues(t, 11, stats.GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetErrorCount(KeaDaemonDHCPv4))
}

// Test updating an error count for a daemon and tracking the communication
// error transition based on the updated count.
func TestUpdateKeaErrorCount(t *testing.T) {
	stats := NewKeaAppCommErrorStats()
	require.NotNil(t, stats)
	require.Equal(t, CommErrorNone, stats.UpdateErrorCount(KeaDaemonCA, 0))
	require.Equal(t, CommErrorNew, stats.UpdateErrorCount(KeaDaemonCA, 2))
	require.Equal(t, CommErrorContinued, stats.UpdateErrorCount(KeaDaemonCA, 3))
	require.Equal(t, CommErrorReset, stats.UpdateErrorCount(KeaDaemonCA, 0))
}

// Test resetting an error count for a selected daemon to 0.
func TestResetKeaErrorCount(t *testing.T) {
	stats := NewKeaAppCommErrorStats()
	stats.ResetErrorCount(KeaDaemonD2)
	require.Zero(t, stats.GetErrorCount(KeaDaemonD2))
	require.EqualValues(t, 10, stats.IncreaseErrorCountBy(KeaDaemonD2, 10))
	require.EqualValues(t, 10, stats.GetErrorCount(KeaDaemonD2))
	stats.ResetErrorCount(KeaDaemonD2)
	require.Zero(t, stats.GetErrorCount(KeaDaemonD2))
}

// Test instantiating a structure holding BIND 9 communication errors.
func TestNewBind9AppCommErrorStats(t *testing.T) {
	state := NewBind9AppCommErrorStats()
	require.NotNil(t, state)
	require.NotNil(t, state.errorCounts)
}

// Test increases an error count for a selected BIND 9 channel type by 1.
func TestIncreaseBind9ErrorCount(t *testing.T) {
	stats := NewBind9AppCommErrorStats()
	require.NotNil(t, stats)
	require.Zero(t, stats.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 1, stats.IncreaseErrorCount(Bind9ChannelRNDC))
	require.EqualValues(t, 1, stats.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 2, stats.IncreaseErrorCount(Bind9ChannelRNDC))
	require.EqualValues(t, 2, stats.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 1, stats.IncreaseErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 2, stats.GetErrorCount(Bind9ChannelRNDC))
	require.EqualValues(t, 1, stats.GetErrorCount(Bind9ChannelStats))
}

// Test increasing an error count for a selected BIND 9 channel type by
// an arbitrary number.
func TestIncreaseBind9ErrorCountBy(t *testing.T) {
	stats := NewBind9AppCommErrorStats()
	require.NotNil(t, stats)
	require.EqualValues(t, 5, stats.IncreaseErrorCountBy(Bind9ChannelRNDC, 5))
	require.EqualValues(t, 5, stats.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 11, stats.IncreaseErrorCountBy(Bind9ChannelRNDC, 6))
	require.EqualValues(t, 11, stats.GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
}

// Resets an error count for a selected channel type to 0.
func TestResetBind9ErrorCount(t *testing.T) {
	stats := NewBind9AppCommErrorStats()
	stats.ResetErrorCount(Bind9ChannelStats)
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
	require.EqualValues(t, 10, stats.IncreaseErrorCountBy(Bind9ChannelStats, 10))
	require.EqualValues(t, 10, stats.GetErrorCount(Bind9ChannelStats))
	stats.ResetErrorCount(Bind9ChannelStats)
	require.Zero(t, stats.GetErrorCount(Bind9ChannelStats))
}

// Test instantiating communication statistics for a new agent.
func TestNewAgentStats(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	require.NotNil(t, stats.agentCommErrors)
	require.NotNil(t, stats.bind9CommErrors)
	require.NotNil(t, stats.keaCommErrors)
	require.NotNil(t, stats.mutex)
}

// Returns a mutex to be used when the data is accessed or updated.
// A caller is responsible for locking and unlocking the mutex.
func (stats *AgentCommStats) TestGetMutex() *sync.RWMutex {
	return stats.mutex
}

// Test increasing communication error count with an agent by 1.
func TestIncreaseAgentErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	require.EqualValues(t, 1, stats.IncreaseErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 1, stats.GetErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 2, stats.IncreaseErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 2, stats.GetErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 3, stats.IncreaseErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 3, stats.GetErrorCount(&requestTypeFoo{}))
	require.Zero(t, stats.GetErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 1, stats.IncreaseErrorCount(&requestTypeBar{}))
	require.EqualValues(t, 3, stats.GetErrorCount(&requestTypeFoo{}))
	require.EqualValues(t, 1, stats.GetErrorCount(&requestTypeBar{}))
}

// Test resetting the communication error count with an agent to 0.
func TestResetErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	for i := 0; i < 10; i++ {
		require.EqualValues(t, i+1, stats.IncreaseErrorCount(requestTypeFoo{}))
	}
	require.EqualValues(t, 10, stats.GetErrorCount(requestTypeFoo{}))
	require.Zero(t, stats.GetErrorCount(requestTypeBar{}))

	stats.ResetErrorCount(requestTypeFoo{})
	require.Zero(t, stats.GetErrorCount(requestTypeFoo{}))
	require.Zero(t, stats.GetErrorCount(requestTypeBar{}))
}

// Test returning a communication error count with an agent for all
// gRPC request types.
func TestGetTotalErrorCount(t *testing.T) {
	stats := NewAgentStats()
	require.NotNil(t, stats)
	for i := 0; i < 10; i++ {
		require.EqualValues(t, i+1, stats.IncreaseErrorCount(requestTypeFoo{}))
	}
	for i := 0; i < 30; i++ {
		require.EqualValues(t, i+1, stats.IncreaseErrorCount(requestTypeBar{}))
	}
	require.EqualValues(t, 10, stats.GetErrorCount(requestTypeFoo{}))
	require.EqualValues(t, 30, stats.GetErrorCount(requestTypeBar{}))
	require.EqualValues(t, 40, stats.GetTotalErrorCount())
}

// Test returning Kea communication error stats for a selected app by ID.
func TestGetKeaCommErrorStats(t *testing.T) {
	stats := NewAgentStats()
	keaStats := stats.GetKeaCommErrorStats(1)
	require.NotNil(t, keaStats)
	require.EqualValues(t, 1, keaStats.IncreaseErrorCount(KeaDaemonCA))
	require.EqualValues(t, 1, stats.GetKeaCommErrorStats(1).GetErrorCount(KeaDaemonCA))
	require.Zero(t, stats.GetKeaCommErrorStats(2).GetErrorCount(KeaDaemonCA))
}

// Test returning BIND 9 communication error stats for a selected app by ID.
func TestGetBind9CommErrorStats(t *testing.T) {
	stats := NewAgentStats()
	bind9Stats := stats.GetBind9CommErrorStats(1)
	require.NotNil(t, bind9Stats)
	require.EqualValues(t, 1, bind9Stats.IncreaseErrorCount(Bind9ChannelRNDC))
	require.EqualValues(t, 1, stats.GetBind9CommErrorStats(1).GetErrorCount(Bind9ChannelRNDC))
	require.Zero(t, stats.GetBind9CommErrorStats(2).GetErrorCount(Bind9ChannelRNDC))
}

// Test that returned statistics is locked and can be unlocked with the
// call to the Close function.
func TestAgentCommStatsWrapperLock(t *testing.T) {
	stats := NewAgentStats()
	wrapper := NewAgentCommStatsWrapper(stats)
	require.NotNil(t, wrapper)
	mutex := wrapper.GetStats().mutex
	// The mutext should be locked for updates. This attempt should
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
	wrapper := NewAgentCommStatsWrapper(stats)
	require.NotNil(t, wrapper)
	defer wrapper.Close()
	returnedStats := wrapper.GetStats()
	require.Equal(t, stats, returnedStats)
}
