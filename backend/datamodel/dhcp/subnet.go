package dhcpmodel

// A common interface describing a subnet in Stork. It defines the
// functions used to retrieve the generic subnet information, and can
// be used to convert the Stork-specific subnet data structures to the
// implementation-specific subnet data structures (e.g., the Kea subnets).
// If we ever integrate Stork with other DHCP server implementations,
// this interface must not be extended with the parameters specific to
// these implementations. Specialized interfaces must be created in the
// appcfg directory, embedding this interface.
type SubnetAccessor interface {
	// Returns a subnet prefix.
	GetPrefix() string
	// Returns a slice of interfaces representing address pools configured
	// for a subnet on a DHCP server.
	GetAddressPools(int64) []AddressPoolAccessor
	// Returns a slice of interfaces representing delegated prefix pools
	// configured for a subnet on a DHCP server.
	GetPrefixPools(int64) []PrefixPoolAccessor
	// Returns a slice of DHCP options configured for a selected daemon in
	// the subnet.
	GetDHCPOptions(int64) []DHCPOptionAccessor
	// Returns a user-context for a selected daemon in the subnet.
	GetUserContext(int64) map[string]any
}
