package keaconfig

import (
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// An interface representing a delegated prefix pool in Stork, extended
// with a set of Kea-specific pool parameters, such as client classes.
type PrefixPool interface {
	dhcpmodel.PrefixPoolAccessor
	GetKeaParameters() *PoolParameters
}

// Represents prefix delegation pool structure within Kea configuration.
type PDPool struct {
	Prefix            string             `json:"prefix"`
	PrefixLen         int                `json:"prefix-len"`
	DelegatedLen      int                `json:"delegated-len"`
	ExcludedPrefix    string             `json:"excluded-prefix,omitempty"`
	ExcludedPrefixLen int                `json:"excluded-prefix-len,omitempty"`
	PoolID            *int64             `json:"pool-id,omitempty"`
	OptionData        []SingleOptionData `json:"option-data,omitempty"`
	ClientClassParameters
}

// Returns a delegated prefix pool in a canonical form.
func (p PDPool) GetCanonicalPrefix() string {
	if p.Prefix != "" && p.PrefixLen != 0 {
		return storkutil.FormatCIDRNotation(p.Prefix, p.PrefixLen)
	}
	return ""
}

// Returns an excluded prefix in a canonical form.
func (p PDPool) GetCanonicalExcludedPrefix() string {
	if p.ExcludedPrefix != "" && p.ExcludedPrefixLen != 0 {
		return storkutil.FormatCIDRNotation(p.ExcludedPrefix, p.ExcludedPrefixLen)
	}
	return ""
}

// Returns a pointer to the pool parameters.
func (p PDPool) GetPoolParameters() *PoolParameters {
	return &PoolParameters{
		PoolID:                p.PoolID,
		ClientClassParameters: p.ClientClassParameters,
	}
}
