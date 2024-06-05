package restservice

// A REST API endpoint operation.
// It is used internally by the REST API handler to detect if a specific
// operation on the REST API endpoint has been disabled by the user.
type EndpointOp int

// An operation to create/register a new machine.
const EndpointOpCreateNewMachine EndpointOp = iota

// Holds a collection of selected endpoint operations' states.
//
// Some REST API operations can be explicitly disabled by a user. This
// structure allows for checking if a particular operation has been disabled.
// By default all operations are enabled.
//
// An operation can refer to the entire endpoint or some particular logic
// within the endpoint handler.
type EndpointControl struct {
	disabledState map[EndpointOp]bool
}

// Instantiates the EndpointControl.
func NewEndpointControl() *EndpointControl {
	return &EndpointControl{
		disabledState: make(map[EndpointOp]bool),
	}
}

// Checks if the specified endpont operation is disabled.
func (ctl *EndpointControl) IsDisabled(endpointType EndpointOp) bool {
	return ctl.disabledState[endpointType]
}

// Sets the endpoint operation state to enabled or disabled.
func (ctl *EndpointControl) SetEnabled(endpointType EndpointOp, enabled bool) {
	ctl.disabledState[endpointType] = !enabled
}
