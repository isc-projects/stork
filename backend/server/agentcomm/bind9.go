package agentcomm

// Store BIND 9 access control configuration.
type Bind9Control struct {
	CtrlAddress string
	CtrlPort    int64
	CtrlKey     string
}
