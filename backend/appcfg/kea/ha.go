package keaconfig

// A structure reflecting an array of high availability configurations
// for a Kea server. It is a top level HA library configuration.
type HALibraryParams struct {
	HA []HA `json:"high-availability"`
}

// A structure representing a single high availability configuration for
// a Kea server. It defines relations between several connected peers
// (e.g, primary, standby and a backup).
type HA struct {
	ThisServerName    *string           `json:"this-server-name"`
	Mode              *string           `json:"mode"`
	HeartbeatDelay    *int              `json:"heartbeat-delay"`
	MaxResponseDelay  *int              `json:"max-response-delay"`
	MaxAckDelay       *int              `json:"max-ack-delay"`
	MaxUnackedClients *int              `json:"max-unacked-clients"`
	Peers             []Peer            `json:"peers"`
	MultiThreading    *HAMultiThreading `json:"multi-threading"`
}

// A structure representing the multi-threading configuration in the
// high availability hook library.
type HAMultiThreading struct {
	EnableMultiThreading  *bool `json:"enable-multi-threading"`
	HTTPDedicatedListener *bool `json:"http-dedicated-listener"`
	HTTPListenerThreads   *int  `json:"http-listener-threads"`
	HTTPClientThreads     *int  `json:"http-client-threads"`
}

// A structure representing one of the peers in the high avalability
// configuration (e.g., a standby server).
type Peer struct {
	Name         *string `json:"name"`
	URL          *string `json:"url"`
	Role         *string `json:"role"`
	AutoFailover *bool   `json:"auto-failover"`
}

// Convenience function returning the first HA configuration.
func (params HALibraryParams) GetFirstRelationship() *HA {
	if len(params.HA) > 0 {
		return &params.HA[0]
	}
	return &HA{}
}

// Returns configurations of all HA relationships.
func (params HALibraryParams) GetAllRelationships() []HA {
	return params.HA
}

// Checks if the mandatory Kea HA configuration parameters are set. It doesn't
// check parameters consistency, though.
func (c HA) IsValid() bool {
	// Check if peers are valid.
	for _, p := range c.Peers {
		if !p.IsValid() {
			return false
		}
	}
	// Check other required parameters.
	return c.ThisServerName != nil && c.Mode != nil
}

// Checks if the mandatory peer parameters are set. It doesn't check if the
// values are correct.
func (p Peer) IsValid() bool {
	return p.Name != nil && p.URL != nil && p.Role != nil
}
