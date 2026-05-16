package dbmodel

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	storktest "isc.org/stork/server/test"
)

// Test that Utilization can be serialized and deserialized correctly.
func TestUtilizationSerializeDeserialize(t *testing.T) {
	// Arrange
	origin := Utilization(0.123456789)

	// Act
	serialized, serializeErr := origin.AppendValue([]byte{}, 0)
	var deserialized Utilization
	reader := storktest.NewPoolReaderMock(serialized, nil)
	deserializeErr := deserialized.ScanValue(reader, len(serialized))

	// Assert
	require.NoError(t, serializeErr)
	require.NoError(t, deserializeErr)
	require.Equal(t, []byte("123"), serialized)
	require.InDelta(t, 0.123, float64(deserialized), 1e-5)
}

// Test that Utilization can be serialized and deserialized with quotes
// correctly.
func TestUtilizationSerializeDeserializeQuotes(t *testing.T) {
	// Arrange
	origin := Utilization(0.123456789)

	// Act
	serialized, serializeErr := origin.AppendValue([]byte{}, 1)
	var deserialized Utilization
	unquoted := []byte(strings.Trim(string(serialized), "'"))
	reader := storktest.NewPoolReaderMock(unquoted, nil)
	deserializeErr := deserialized.ScanValue(reader, len(serialized))

	// Assert
	require.NoError(t, serializeErr)
	require.NoError(t, deserializeErr)
	require.Equal(t, []byte("'123'"), serialized)
	require.InDelta(t, 0.123, float64(deserialized), 1e-5)
}

// Test that the serialized utilization value is clipped to the 2-byte smallint
// limits when the utilization exceeds the bounds.
func TestUtilizationExceedsBounds(t *testing.T) {
	// Arrange
	above := Utilization(math.MaxInt16 + 1e-7)
	below := Utilization(math.MinInt16 - 1e-7)

	// Act
	aboveSerialized, aboveSerializeErr := above.AppendValue([]byte{}, 0)
	belowSerialized, belowSerializeErr := below.AppendValue([]byte{}, 0)

	// Assert
	require.NoError(t, aboveSerializeErr)
	require.NoError(t, belowSerializeErr)
	require.Equal(t, []byte(strconv.FormatInt(math.MaxInt16, 10)), aboveSerialized)
	require.Equal(t, []byte(strconv.FormatInt(math.MinInt16, 10)), belowSerialized)

}
