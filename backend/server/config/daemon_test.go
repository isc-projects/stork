package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test DaemonTag interface implementation.
func TestDaemonTag(t *testing.T) {
	tag := newDaemonTag(Daemon{
		ID:    1,
		Name:  "dhcp4",
		AppID: 2,
	}, 42)
	require.EqualValues(t, 1, tag.GetID())
	require.Equal(t, "dhcp4", tag.GetName())
	require.EqualValues(t, 2, tag.GetAppID())
	require.Equal(t, dbmodel.AppTypeKea, tag.GetAppType())
	require.NotNil(t, tag.GetMachineID())
	require.EqualValues(t, 42, *tag.GetMachineID())
}

// Test that GetAppType() returns "kea" for Kea daemons.
func TestDaemonTagKeaAppType(t *testing.T) {
	names := []string{
		dbmodel.DaemonNameDHCPv4,
		dbmodel.DaemonNameDHCPv6,
		dbmodel.DaemonNameCA,
		dbmodel.DaemonNameD2,
	}
	for _, name := range names {
		daemonName := name
		t.Run(name, func(t *testing.T) {
			tag := newDaemonTag(Daemon{
				Name: daemonName,
			}, 42)
			require.Equal(t, dbmodel.AppTypeKea, tag.GetAppType())
		})
	}
}

// Test that GetAppType() returns "bind9" for daemon name "named".
func TestDaemonTagBind9AppType(t *testing.T) {
	tag := newDaemonTag(Daemon{
		Name: "named",
	}, 42)
	require.Equal(t, dbmodel.AppTypeBind9, tag.GetAppType())
}

// Test that GetAppType() returns "unknown" for unsupported daemon name.
func TestDaemonTagUnknownApp(t *testing.T) {
	daemon := newDaemonTag(Daemon{
		Name: "something",
	}, 42)
	require.Equal(t, "unknown", daemon.GetAppType())
}
