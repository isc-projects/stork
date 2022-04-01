package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test DaemonTag interface implementation.
func TestDaemonTag(t *testing.T) {
	daemon := Daemon{
		ID:    1,
		Name:  "dhcp4",
		AppID: 2,
	}
	require.EqualValues(t, 1, daemon.GetID())
	require.Equal(t, "dhcp4", daemon.GetName())
	require.EqualValues(t, 2, daemon.GetAppID())
	require.Equal(t, dbmodel.AppTypeKea, daemon.GetAppType())
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
			daemon := Daemon{
				Name: daemonName,
			}
			require.Equal(t, dbmodel.AppTypeKea, daemon.GetAppType())
		})
	}
}

// Test that GetAppType() returns "bind9" for daemon name "named".
func TestDaemonTagBind9AppType(t *testing.T) {
	daemon := Daemon{
		Name: "named",
	}
	require.Equal(t, dbmodel.AppTypeBind9, daemon.GetAppType())
}

// Test that GetAppType() returns "unknown" for unsupported daemon name.
func TestDaemonTagUnknownApp(t *testing.T) {
	daemon := Daemon{
		Name: "something",
	}
	require.Equal(t, "unknown", daemon.GetAppType())
}
