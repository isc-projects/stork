package agentcomm

// Store BIND 9 access control configuration.
type Bind9Control struct {
	Address string
	Port    int64
	Key     string
}
