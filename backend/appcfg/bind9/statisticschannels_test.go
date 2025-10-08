package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting the first inet clause from the statistics-channels statement.
func TestStatisticsChannelsGetFirstInetClause(t *testing.T) {
	statisticsChannels := &StatisticsChannels{
		Clauses: []*InetClause{
			{
				Address: "127.0.0.1",
				Port:    storkutil.Ptr("953"),
			},
			{
				Address: "192.0.2.1",
				Port:    storkutil.Ptr("8053"),
			},
			{
				Address: "::1",
				Port:    storkutil.Ptr("8054"),
			},
		},
	}
	inetClause := statisticsChannels.GetFirstInetClause()
	require.NotNil(t, inetClause)
	require.Equal(t, statisticsChannels.Clauses[0], inetClause)
	require.Equal(t, "127.0.0.1", inetClause.Address)
	require.EqualValues(t, "953", *inetClause.Port)
}

// Test that nil is returned when getting the inet clause from the statistics-channels statement
// when the statistics-channels statement is empty.
func TestStatisticsChannelsGetFirstInetClauseNone(t *testing.T) {
	statisticsChannels := &StatisticsChannels{}
	require.Nil(t, statisticsChannels.GetFirstInetClause())
}
