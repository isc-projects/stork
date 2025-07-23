package dnsconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that a valid RR is parsed correctly.
func TestNewRR(t *testing.T) {
	rr, err := NewRR("example.com. 3600 IN A 192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, "example.com.", rr.Name)
	require.EqualValues(t, 3600, rr.TTL)
	require.Equal(t, "A", rr.Type)
	require.Equal(t, "IN", rr.Class)
	require.Equal(t, "192.0.2.1", rr.Rdata)
}

// Test that an RR with multiple Rdata fields is parsed correctly.
func TestNewRRMultipleRdataFields(t *testing.T) {
	rr, err := NewRR("example.com. 120 SOA ns1.example.com. hostmaster.example.com. 2025071700 1800 900 604800 86400")
	require.NoError(t, err)
	require.Equal(t, "example.com.", rr.Name)
	require.EqualValues(t, 120, rr.TTL)
	require.Equal(t, "SOA", rr.Type)
	require.Equal(t, "IN", rr.Class)
	require.Equal(t, "ns1.example.com. hostmaster.example.com. 2025071700 1800 900 604800 86400", rr.Rdata)
}

// Test that an RR with no Rdata fields is parsed correctly.
func TestNewRRNoRdataFields(t *testing.T) {
	rr, err := NewRR("example.com. 120 IN NULL")
	require.NoError(t, err)
	require.Equal(t, "example.com.", rr.Name)
	require.EqualValues(t, 120, rr.TTL)
	require.Equal(t, "NULL", rr.Type)
	require.Equal(t, "IN", rr.Class)
	require.Equal(t, "", rr.Rdata)
}

// Test that an error is returned when an invalid RR is parsed.
func TestNewRRInvalidRR(t *testing.T) {
	rr, err := NewRR("example.com. 3600 IN")
	require.Error(t, err)
	require.Nil(t, rr)
	require.ErrorContains(t, err, "failed to parse RR: example.com. 3600 IN")
}

// Test that the string representation of an RR is correct.
func TestNewRRGetString(t *testing.T) {
	rr, err := NewRR("example.com. 3600 IN A 192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, "example.com. 3600 IN A 192.0.2.1", rr.GetString())
}
