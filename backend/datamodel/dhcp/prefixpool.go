package dhcpmodel

import (
	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// A common interface describing a delegated prefix pool in Stork. It defines the
// functions used to retrieve the generic pool information, and can be used to
// convert the Stork-specific pool data structures to the implementation-specific
// pool data structures (e.g., the Kea address pools). If we ever integrate
// Stork with other DHCP server implementations, this interface must not be
// extended with the parameters specific to these implementations. Specialized
// interfaces must be created in the appcfg directory, embedding this interface.
// See keaconfig.PrefixPool interface.
type PrefixPoolAccessor interface {
	// Returns a pointer to a structure holding the delegated prefix data.
	GetModel() *PrefixPool
	// Returns a slice of interfaces describing the DHCP options for a pool.
	GetDHCPOptions() []DHCPOptionAccessor
}

// A structure holding the delegated prefix information. It includes the delegated
// prefix, delegated prefix length and the excluded prefix (see RFC 6603). This
// structure exposes the convenience functions returning the prefixes and their
// lengths separately, rather than in the canonical form. The prefixes are stored
// in the canonical form in the database. These functions make it convenient to
// convert the prefixes from the database format to other formats, if necessary.
type PrefixPool struct {
	Prefix         string
	DelegatedLen   int
	ExcludedPrefix string
}

// Converts a delegated prefix from the canonical form to the prefix/length
// tuple. It returns an error if the prefix is invalid.
func (p *PrefixPool) GetPrefix() (string, int, error) {
	parsedIP := storkutil.ParseIP(p.Prefix)
	if parsedIP == nil || !parsedIP.Prefix {
		return "", 0, errors.Errorf("invalid prefix %s", p.Prefix)
	}
	return parsedIP.NetworkPrefix, parsedIP.PrefixLength, nil
}

// Converts an excluded prefix from the canonical form to the prefix/length
// tuple. It returns an error if the prefix is invalid.
func (p *PrefixPool) GetExcludedPrefix() (string, int, error) {
	if len(p.ExcludedPrefix) == 0 {
		return "", 0, nil
	}
	parsedIP := storkutil.ParseIP(p.ExcludedPrefix)
	if parsedIP == nil || !parsedIP.Prefix {
		return "", 0, errors.Errorf("invalid excluded prefix %s", p.ExcludedPrefix)
	}
	return parsedIP.NetworkPrefix, parsedIP.PrefixLength, nil
}
