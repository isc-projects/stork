package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test MachineTag interface implementation.
func TestMachineTag(t *testing.T) {
	machine := Machine{
		ID:      10,
		Address: "192.0.2.2",
		State: MachineState{
			Hostname: "cool.example.org",
		},
	}
	require.EqualValues(t, 10, machine.GetID())
	require.Equal(t, "192.0.2.2", machine.GetAddress())
	require.Equal(t, "cool.example.org", machine.GetHostname())
}
