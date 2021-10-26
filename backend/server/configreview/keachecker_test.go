package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Tests that the checker checking stat_cmds hooks library presence
// returns nil when the library is loaded.
func TestStatCmdsPresent(t *testing.T) {
	t.Run("stat_cmds_present", func(t *testing.T) {
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
	})
}

// Tests that the checker checking stat_cmds hooks library presence
// returns the report when the library is not loaded.
func TestStatCmdsAbsent(t *testing.T) {
	t.Run("stat_cmds_absent", func(t *testing.T) {
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
		require.Contains(t, report.content, "Consider using the libdhcp_stat_cmds")
	})
}
