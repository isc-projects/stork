package agent

import (
	"testing"
	"context"

	"github.com/stretchr/testify/require"

	"isc.org/stork/api"
)


func TestGetVersion(t *testing.T) {
	sa := StorkAgent{}

	ctx := context.Background()
	rsp, err := sa.GetVersion(ctx, &agentapi.Empty{})
	require.NoError(t, err)
	require.Equal(t, *rsp, agentapi.AgentVersion{Version: "1.0.9a"})
}
