package restservice

import (
	"log"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/services"
	"isc.org/stork/server/agentcomm"
)

type FakeAgents struct {
	a string
}

func (fa *FakeAgents) GetSettings() *agentcomm.AgentsSettings {
	return nil
}
func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}
func (fa *FakeAgents) GetState(address string) (*agentcomm.State, error) {
	state := agentcomm.State{
		Cpus: 1,
		Memory: 4,
	}
	return &state, nil
}


func TestCreateMachine(t *testing.T) {
	rapi := RestAPI{}
	fa := FakeAgents{}
	rapi.Init(&dbtest.GenericConnOptions, &fa)
	ctx := context.Background()

	addr := "1.2.3.4"
	params := services.CreateMachineParams{
		Machine: &models.Machine{
			Address: &addr,
		},
	}
	rsp := rapi.CreateMachine(ctx, params)
	m := rsp.(*services.CreateMachineOK).Payload
	log.Printf("RESP: %+v", m)
	require.Equal(t, *m.Address, addr)
	require.Greater(t, m.Memory, int64(0))
	require.Greater(t, m.Cpus, int64(0))
	require.GreaterOrEqual(t, m.Uptime, int64(0))
}

func TestGetMachines(t *testing.T) {
	rapi := RestAPI{}
	ctx := context.Background()

	params := services.GetMachinesParams{
	}

	rsp := rapi.GetMachines(ctx, params)
	ms := rsp.(*services.GetMachinesOK).Payload
	log.Printf("RESP: %+v", ms)
	require.Greater(t, ms.Total, int64(0))
	//require.Greater(t, ms.Items, )
}
