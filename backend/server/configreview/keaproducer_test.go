package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Tests the producer checking stat_cmds hooks library presence works
// as expected.
func TestStatCmdsPresence(t *testing.T) {
	ctx := newReviewContext()

	t.Run("stat_cmds_absent", func(t *testing.T) {
		config, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
		require.NoError(t, err)

		ctx.subjectDaemon = &dbmodel.Daemon{
			ID: 1,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: config,
			},
		}
		report, err := statCmdsPresence(ctx)
		require.NoError(t, err)
		require.NotNil(t, report)
		require.Contains(t, report.issue, "Consider using the libdhcp_stat_cmds")
	})

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

		ctx.subjectDaemon = &dbmodel.Daemon{
			ID: 1,
			KeaDaemon: &dbmodel.KeaDaemon{
				Config: config,
			},
		}
		report, err := statCmdsPresence(ctx)
		require.NoError(t, err)
		require.Nil(t, report)
	})
}
