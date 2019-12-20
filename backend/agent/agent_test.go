package agent

import (
	"testing"
	"context"

	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
	"isc.org/stork"
)

type FakeAppMonitor struct {
	Apps []interface{}
}

func (fsm *FakeAppMonitor) GetApps() []interface{} {
	return nil
}

func (fsm *FakeAppMonitor) Shutdown() {
}


func TestGetState(t *testing.T) {
	fsm := FakeAppMonitor{}
	sa := StorkAgent{
		AppMonitor: &fsm,
	}

	// app monitor is empty, no apps should be returned by GetState
	ctx := context.Background()
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)

	// add some app to app monitor so GetState should return something
	var apps []interface{}
	apps = append(apps, AppKea{
		AppCommon: AppCommon{
			Version: "1.2.3",
			Active: true,
		},
	})
	fsm.Apps = apps
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
}
