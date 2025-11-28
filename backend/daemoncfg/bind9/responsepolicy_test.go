package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking whether or not a zone is RPZ.
func TestIsRPZ(t *testing.T) {
	rp := &ResponsePolicy{
		Zones: []*ResponsePolicyZone{
			{
				Zone: "rpz.example.com",
			},
		},
	}
	require.True(t, rp.IsRPZ("rpz.example.com"))
	require.True(t, rp.IsRPZ("RPZ.Example.Com"))
	require.False(t, rp.IsRPZ("db.local"))
}

// Test that the response-policy statement is formatted correctly.
func TestResponsePolicyFormat(t *testing.T) {
	rp := &ResponsePolicy{
		Zones: []*ResponsePolicyZone{
			{
				Zone:     "rpz.example.com",
				Switches: []string{"max-policy-ttl", "100", "min-update-interval", "102"},
			},
			{
				Zone: "rpz.local",
			},
		},
		Switches: []string{"add-soa", "true", "break-dnssec", "false"},
	}
	output := rp.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterStringBuilder()
	err := output.write(0, false, builder)
	require.NoError(t, err)
	requireConfigEq(t, `response-policy {
		zone "rpz.example.com" max-policy-ttl 100 min-update-interval 102;
		zone "rpz.local";
	} add-soa true break-dnssec false;`, output)
}

// Test that serializing a response-policy statement with nil values does not panic.
func TestResponsePolicyFormatNilValues(t *testing.T) {
	rp := &ResponsePolicy{}
	require.NotPanics(t, func() { rp.getFormattedOutput(nil) })
}

// Test that serializing a response-policy zone with nil values does not panic.
func TestResponsePolicyZoneFormatNilValues(t *testing.T) {
	rpz := &ResponsePolicyZone{}
	require.NotPanics(t, func() { rpz.getFormattedOutput(nil) })
}
