package agent

import (
	"testing"
	"context"

	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
	"isc.org/stork"
)

type FakeServiceMonitor struct {
	Services []interface{}
}

func (fsm *FakeServiceMonitor) GetServices() []interface{} {
	return nil
}

func (fsm *FakeServiceMonitor) Shutdown() {
}


func TestGetState(t *testing.T) {
	fsm := FakeServiceMonitor{}
	sa := StorkAgent{
		ServiceMonitor: &fsm,
	}

	// service monitor is empty, no services should be returned by GetState
	ctx := context.Background()
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)

	// add some service to service monitor so GetState should return something
	var services []interface{}
	services = append(services, ServiceKea{
		ServiceCommon: ServiceCommon{
			Version: "1.2.3",
			Active: true,
		},
	})
	fsm.Services = services
	rsp, err = sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
}
