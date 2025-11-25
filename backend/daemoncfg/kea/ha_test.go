package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Checks if the HA peer structure validation works as expected.
func TestPeerParametersValid(t *testing.T) {
	p := Peer{}
	require.False(t, p.IsValid())

	name := "server1"
	p.Name = &name
	require.False(t, p.IsValid())

	url := "http://example.org/"
	p.URL = &url
	require.False(t, p.IsValid())

	role := "primary"
	p.Role = &role
	require.True(t, p.IsValid())

	autoFailover := true
	p.AutoFailover = &autoFailover
	require.True(t, p.IsValid())
}

// Checks if the HA configuration validation works as expected.
func TestHAConfigParametersValid(t *testing.T) {
	cfg := HA{}

	require.False(t, cfg.IsValid())

	thisServerName := "server1"
	cfg.ThisServerName = &thisServerName
	require.False(t, cfg.IsValid())

	haMode := "load-balancing"
	cfg.Mode = &haMode
	require.True(t, cfg.IsValid())

	p := Peer{}
	cfg.Peers = append(cfg.Peers, p)
	require.False(t, cfg.IsValid())
}

// Test getting all HA relationships from the HA library params.
func TestGetAllRelationships(t *testing.T) {
	cfg := HALibraryParams{
		HA: []HA{
			{
				ThisServerName: storkutil.Ptr("server1"),
			},
			{
				ThisServerName: storkutil.Ptr("server2"),
			},
		},
	}
	relationships := cfg.GetAllRelationships()
	require.Len(t, relationships, 2)
	require.Equal(t, "server1", *relationships[0].ThisServerName)
	require.Equal(t, "server2", *relationships[1].ThisServerName)
}

// Test that MT is by default disabled in the Kea versions earlier
// than 2.3.7.
func TestIsMultiThreadingEnabledDefault235(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
	}
	require.False(t, ha.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 5)))
}

// Test that MT can be explicitly enabled in the Kea versions earlier
// than 2.3.7.
func TestIsMultiThreadingEnabledEnabled235(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
		MultiThreading: &HAMultiThreading{
			EnableMultiThreading: storkutil.Ptr(true),
		},
	}
	require.True(t, ha.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 5)))
}

// Test that MT is by default enabled in the Kea versions later than
// or equal 2.3.7.
func TestIsMultiThreadingEnabledDefault238(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
	}
	require.True(t, ha.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 8)))
}

// Test that MT can be explicitly disabled in the Kea versions later than
// or equal 2.3.7.
func TestIsMultiThreadingEnabledDisabled238(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
		MultiThreading: &HAMultiThreading{
			EnableMultiThreading: storkutil.Ptr(false),
		},
	}
	require.False(t, ha.IsMultiThreadingEnabled(storkutil.NewSemanticVersion(2, 3, 8)))
}

// Test that HTTP dedicated listener is by default disabled in the Kea
// versions earlier than 2.3.7.
func TestIsDedicatedListenerEnabledDefault235(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
	}
	require.False(t, ha.IsDedicatedListenerEnabled(storkutil.NewSemanticVersion(2, 3, 5)))
}

// Test that HTTP dedicated listener can be explicitly enabled in the Kea
// versions earlier than 2.3.7.
func TestIsDedicatedListenerEnabledEnabled235(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
		MultiThreading: &HAMultiThreading{
			HTTPDedicatedListener: storkutil.Ptr(true),
		},
	}
	require.True(t, ha.IsDedicatedListenerEnabled(storkutil.NewSemanticVersion(2, 3, 5)))
}

// Test that HTTP dedicated listener is by default enabled in the Kea
// versions later than or equal 2.3.7.
func TestIsMultiThreadingEnabledDefaults238(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
	}
	require.True(t, ha.IsDedicatedListenerEnabled(storkutil.NewSemanticVersion(2, 3, 8)))
}

// Test that HTTP dedicated listener can be explicitly disabled in the Kea
// versions later than or equal 2.3.7.
func TestIsDedicatedListenerEnabledDisabled238(t *testing.T) {
	ha := HA{
		ThisServerName: storkutil.Ptr("server1"),
		MultiThreading: &HAMultiThreading{
			HTTPDedicatedListener: storkutil.Ptr(false),
		},
	}
	require.False(t, ha.IsDedicatedListenerEnabled(storkutil.NewSemanticVersion(2, 3, 8)))
}
