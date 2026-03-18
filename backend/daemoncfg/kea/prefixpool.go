package keaconfig

import (
	"encoding/json"

	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// An interface representing a delegated prefix pool in Stork, extended
// with a set of Kea-specific pool parameters, such as client classes.
type PrefixPool interface {
	dhcpmodel.PrefixPoolAccessor
	GetKeaParameters() *PoolParameters
}

// Represents known (supported by Stork) configuration parameters for a delegated prefix pool.
type PDPoolKnownParameters struct {
	Prefix            string             `json:"prefix"`
	PrefixLen         int                `json:"prefix-len"`
	DelegatedLen      int                `json:"delegated-len"`
	ExcludedPrefix    string             `json:"excluded-prefix,omitempty"`
	ExcludedPrefixLen int                `json:"excluded-prefix-len,omitempty"`
	PoolID            int64              `json:"pool-id,omitempty"`
	OptionData        []SingleOptionData `json:"option-data,omitempty"`
	ClientClassParameters
}

// Represents prefix delegation pool structure within Kea configuration.
type PDPool struct {
	PDPoolKnownParameters
	UnknownParameters map[string]any `json:"-"`
}

// Unmarshals the JSON data into the PDPool structure. The output contains
// the known parameters and a map of unknown parameters.
func (p *PDPool) UnmarshalJSON(data []byte) error {
	poolWithUnknown := WithUnknown[PDPoolKnownParameters]{}
	if err := json.Unmarshal(data, &poolWithUnknown); err != nil {
		return err
	}
	*p = PDPool{
		PDPoolKnownParameters: poolWithUnknown.Known,
		UnknownParameters:     poolWithUnknown.Unknown,
	}
	return nil
}

// Marshals the PDPool structure into JSON. The output contains the known
// parameters and a map of unknown parameters.
func (p PDPool) MarshalJSON() ([]byte, error) {
	poolWithUnknown := WithUnknown[PDPoolKnownParameters]{
		Known:   p.PDPoolKnownParameters,
		Unknown: p.UnknownParameters,
	}
	return json.Marshal(poolWithUnknown)
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
		UnknownParameters:     p.UnknownParameters,
	}
}
