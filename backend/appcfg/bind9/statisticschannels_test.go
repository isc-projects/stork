package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test getting the first inet clause from the statistics-channels statement.
func TestStatisticsChannelsGetInetClause(t *testing.T) {
	statisticsChannels := &StatisticsChannels{
		Clauses: []*InetClause{
			{
				Address: "127.0.0.1",
				Port:    storkutil.Ptr("953"),
			},
		},
	}
	inetClause := statisticsChannels.GetInetClause()
	require.NotNil(t, inetClause)
	require.Equal(t, statisticsChannels.Clauses[0], inetClause)
	require.Equal(t, "127.0.0.1", inetClause.Address)
	require.EqualValues(t, "953", *inetClause.Port)
}

// Test that nil is returned when getting the inet clause from the statistics-channels statement
// when the statistics-channels statement is empty.
func TestStatisticsChannelsGetInetClauseNone(t *testing.T) {
	statisticsChannels := &StatisticsChannels{}
	require.Nil(t, statisticsChannels.GetInetClause())
}
