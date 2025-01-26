package agentcomm

import "fmt"

// An error created when zone inventory on an agent was running
// a long lasting operation while trying to fetch zones from it.
type ZoneInventoryBusyError struct {
	agent string
}

// Instantiates the ZoneInventoryBusyError.
func NewZoneInventoryBusyError(agent string) *ZoneInventoryBusyError {
	return &ZoneInventoryBusyError{
		agent,
	}
}

// Returns an error string.
func (err *ZoneInventoryBusyError) Error() string {
	return fmt.Sprintf("Zone inventory is temporarily busy on the agent %s", err.agent)
}

// An error created when zone inventory on an agent was not
// initialized while trying to fetch zones from it.
type ZoneInventoryNotInitedError struct {
	agent string
}

// Instantiates the ZoneInventoryNotInitedError.
func NewZoneInventoryNotInitedError(agent string) *ZoneInventoryNotInitedError {
	return &ZoneInventoryNotInitedError{
		agent,
	}
}

// Returns an error string.
func (err *ZoneInventoryNotInitedError) Error() string {
	return fmt.Sprintf("DNS zones have not been loaded on the agent %s", err.agent)
}
