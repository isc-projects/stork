package dhcpmodel

// A common interface describing an address pool in Stork. It defines the
// functions used to retrieve the generic pool information, and can be used to
// convert the Stork-specific pool data structures to the implementation-specific
// pool data structures (e.g., the Kea address pools). If we ever integrate
// Stork with other DHCP server implementations, this interface must not be
// extended with the parameters specific to these implementations. Specialized
// interfaces must be created in the appcfg directory, embedding this interface.
// See keaconfig.AddressPool interface.
type AddressPoolAccessor interface {
	// Returns lower pool boundary.
	GetLowerBound() string
	// Returns upper pool boundary.
	GetUpperBound() string
	// Returns a slice of interfaces describing the DHCP options for a pool.
	GetDHCPOptions() []DHCPOptionAccessor
}
