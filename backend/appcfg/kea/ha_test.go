package keaconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
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
