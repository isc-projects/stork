package keactrl

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestNewStatus(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		// Arrange
		statusStr := `{
			"csv-lease-file": "/tmp/kea-leases6.csv"
		}`

		// Act
		status, err := NewStatus([]byte(statusStr))

		// Assert
		require.NoError(t, err)
		require.NotNil(t, status)
		require.NotNil(t, status.CSVLeaseFile)
		require.Equal(t, "/tmp/kea-leases6.csv", *status.CSVLeaseFile)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		// Arrange
		// Invalid because of the trailing comma
		statusStr := `{
			"csv-lease-file": "/tmp/kea-leases6.csv",
		}`

		// Act
		_, err := NewStatus([]byte(statusStr))

		// Assert
		require.ErrorContains(t, err, "invalid")
	})
}
