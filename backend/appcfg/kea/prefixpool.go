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
	Prefix               string
	PrefixLen            int                `mapstructure:"prefix-len" json:"prefix-len"`
	DelegatedLen         int                `mapstructure:"delegated-len" json:"delegated-len"`
	ExcludedPrefix       string             `mapstructure:"excluded-prefix" json:"excluded-prefix,omitempty"`
	ExcludedPrefixLen    int                `mapstructure:"excluded-prefix-len" json:"excluded-prefix-len,omitempty"`
	ClientClass          string             `mapstructure:"client-class" json:"client-class,omitempty"`
	RequireClientClasses []string           `mapstructure:"require-client-classes" json:"require-client-classes,omitempty"`
	OptionData           []SingleOptionData `mapstructure:"option-data" json:"option-data,omitempty"`
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
