package keaconfig

import storkutil "isc.org/stork/util"

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

// A structure representing one of the peers in the high availability
// configuration (e.g., a standby server).
type Peer struct {
	Name         *string `json:"name"`
	URL          *string `json:"url"`
	Role         *string `json:"role"`
	AutoFailover *bool   `json:"auto-failover"`
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

// Checks if multi threading has been enabled for an HA relationship.
// The Kea version 2.3.7 and later enable multi threading by default.
func (c HA) IsMultiThreadingEnabled(keaVersion storkutil.SemanticVersion) bool {
	if keaVersion.GreaterThanOrEqual(storkutil.NewSemanticVersion(2, 3, 7)) {
		return c.MultiThreading == nil ||
			c.MultiThreading.EnableMultiThreading == nil ||
			*c.MultiThreading.EnableMultiThreading
	}
	return c.MultiThreading != nil &&
		c.MultiThreading.EnableMultiThreading != nil &&
		*c.MultiThreading.EnableMultiThreading
}

// Checks if an HTTP dedicated listener has been enabled for an HA relationship.
// The Kea version 2.3.7 and later enable the dedicated listener by default.
func (c HA) IsDedicatedListenerEnabled(keaVersion storkutil.SemanticVersion) bool {
	if keaVersion.GreaterThanOrEqual(storkutil.NewSemanticVersion(2, 3, 7)) {
		return c.MultiThreading == nil ||
			c.MultiThreading.HTTPDedicatedListener == nil ||
			*c.MultiThreading.HTTPDedicatedListener
	}
	return c.MultiThreading != nil &&
		c.MultiThreading.HTTPDedicatedListener != nil &&
		*c.MultiThreading.HTTPDedicatedListener
}

// Checks if the mandatory peer parameters are set. It doesn't check if the
// values are correct.
func (p Peer) IsValid() bool {
	return p.Name != nil && p.URL != nil && p.Role != nil
}
