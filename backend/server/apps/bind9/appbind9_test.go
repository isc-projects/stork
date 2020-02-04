package bind9

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test"
)

func TestGetAppState(t *testing.T) {
	ctx := context.Background()

	dummyFn := func(c int, r []interface{}) {
	}

	fa := storktest.NewFakeAgents(dummyFn)

	dbApp := dbmodel.App{
		CtrlAddress: "127.0.0.1",
		CtrlPort:    953,
		CtrlKey:     "abcd",
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	GetAppState(ctx, fa, &dbApp)

	require.Equal(t, "127.0.0.1", fa.RecordedCtrlAddress)
	require.Equal(t, int64(953), fa.RecordedCtrlPort)
	require.Equal(t, "abcd", fa.RecordedCtrlKey)

	require.True(t, dbApp.Active)
	require.Equal(t, dbApp.Meta.Version, "9.9.9")

	daemon := dbApp.Details.(dbmodel.AppBind9).Daemon
	require.True(t, daemon.Active)
	require.Equal(t, daemon.Name, "named")
	require.Equal(t, daemon.Version, "9.9.9")
	reloadedAt, _ := time.Parse(namedLongDateFormat, "Mon, 03 Feb 2020 14:39:36 GMT")
	require.Equal(t, daemon.ReloadedAt, reloadedAt)
	require.Equal(t, daemon.ZoneCount, int64(5))
	require.Equal(t, daemon.AutomaticZoneCount, int64(96))
}
