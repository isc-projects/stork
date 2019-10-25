package agent

import (
	"testing"
	"context"

	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
	"isc.org/stork"
)


func TestGetState(t *testing.T) {
	sa := StorkAgent{}

	ctx := context.Background()
	rsp, err := sa.GetState(ctx, &agentapi.GetStateReq{})
	require.NoError(t, err)
	require.Equal(t, rsp.AgentVersion, stork.Version)
}
