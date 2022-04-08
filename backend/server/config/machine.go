package config

// Machine state being a part of the Machine structure.
type MachineState struct {
	Hostname string
}

// An implementation of the dbmodel.MachineTag interface used
// by the configuration manager to represent machines with apps
// to which control commands can be sent.
type Machine struct {
	ID        int64
	Address   string
	AgentPort int64
	State     MachineState
}

// Returns machine ID.
func (machine Machine) GetID() int64 {
	return machine.ID
}

// Returns machine address.
func (machine Machine) GetAddress() string {
	return machine.Address
}

// Returns machine agent port.
func (machine Machine) GetAgentPort() int64 {
	return machine.AgentPort
}

// Returns hostname.
func (machine Machine) GetHostname() string {
	return machine.State.Hostname
}
