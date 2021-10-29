package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Tests that the checker checking stat_cmds hooks library presence
// returns nil when the library is loaded.
func TestStatCmdsPresent(t *testing.T) {
	config, err := dbmodel.NewKeaConfigFromJSON(`
    {
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_stat_cmds.so"
                }
            ]
        }
    }`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := statCmdsPresence(ctx)
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking stat_cmds hooks library presence
// returns the report when the library is not loaded.
func TestStatCmdsAbsent(t *testing.T) {
	config, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := statCmdsPresence(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "The Kea Statistics Commands library")
}

// Tests that the checker checking host_cmds hooks library presence
// returns nil when the library is loaded.
func TestHostCmdsPresent(t *testing.T) {
	// The host backend is in use and the library is loaded.
	config, err := dbmodel.NewKeaConfigFromJSON(`
    {
        "Dhcp4": {
            "hosts-database": [
                {
                    "type": "mysql"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := hostCmdsPresence(ctx)
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds presence takes into
// account whether or not the host-database(s) parameters are
// also specified.
func TestHostCmdsBackendUnused(t *testing.T) {
	// The backend is not used and the library is not loaded.
	// There should be no report.
	config, err := dbmodel.NewKeaConfigFromJSON(`
    {
        "Dhcp4": { }
    }`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := hostCmdsPresence(ctx)
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// host-database (singular) parameter is specified.
func TestHostCmdsAbsentHostsDatabase(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	config, err := dbmodel.NewKeaConfigFromJSON(`
    {
        "Dhcp4": {
            "hosts-database": {
                "type": "mysql"
            }
        }
    }`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := hostCmdsPresence(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// hosts-databases (plural) parameter is specified.
func TestHostCmdsAbsentHostsDatabases(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	config, err := dbmodel.NewKeaConfigFromJSON(`
    {
        "Dhcp4": {
            "hosts-databases": [
                {
                    "type": "mysql"
                }
            ]
        }
    }`)
	require.NoError(t, err)

	ctx := newReviewContext(&dbmodel.Daemon{
		ID: 1,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	report, err := hostCmdsPresence(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}
