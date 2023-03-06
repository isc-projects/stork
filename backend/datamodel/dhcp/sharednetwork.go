package dhcpmodel

// A common interface describing a shared network in Stork. It defines the
// functions used to retrieve the generic shared network information, and can
// be used to convert the Stork-specific network data structures to the
// implementation-specific data structures (e.g., the Kea shared networks).
// If we ever integrate Stork with other DHCP server implementations,
// this interface must not be extended with the parameters specific to
// these implementations. Specialized interfaces must be created in the
// appcfg directory, embedding this interface.
type SharedNetworkAccessor interface {
	// Returns a slice of DHCP options configured for a selected daemon in
	// the subnet.
	GetDHCPOptions(int64) []DHCPOptionAccessor
}
