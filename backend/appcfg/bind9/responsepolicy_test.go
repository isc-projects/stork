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
