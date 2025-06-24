package pdnsdata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test successfully extracting an integer value from a statistic item.
func TestStatisticItemGetInt64(t *testing.T) {
	item := StatisticItem{
		Name:  "uptime",
		Value: "1234",
	}
	require.EqualValues(t, 1234, item.GetInt64())
}

// Test that zero is returned when the value is not a valid integer.
func TestStatisticItemGetInt64InvalidValue(t *testing.T) {
	item := StatisticItem{
		Name:  "uptime",
		Value: "invalid",
	}
	require.Zero(t, item.GetInt64())
}
