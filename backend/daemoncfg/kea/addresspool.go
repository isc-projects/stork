package keaconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// An interface representing an address pool in Stork, extended with a set
// of Kea-specific pool parameters, such as client classes.
type AddressPool interface {
	dhcpmodel.AddressPoolAccessor
	GetKeaParameters() *PoolParameters
}

// A structure holding Kea-specific pool parameters. Note that the same
// set of parameters is supported for the address and delegated prefix
// pools. This structure should be extended with new Kea-specific parameters
// when they are implemented in Kea.
type PoolParameters struct {
	ClientClassParameters
	PoolID            int64 `json:"pool-id,omitempty"`
	UnknownParameters map[string]any
}

// Represents known (supported by Stork) configuration parameters for an address pool.
type PoolKnownParameters struct {
	Pool       string             `json:"pool"`
	PoolID     int64              `json:"pool-id,omitempty"`
	OptionData []SingleOptionData `json:"option-data,omitempty"`
	ClientClassParameters
}

// Represents an address pool structure within a Kea configuration.
type Pool struct {
	PoolKnownParameters
	UnknownParameters map[string]any `json:"-"`
}

// A custom unmarshal function for a Kea address pool. It removes whitespaces from
// the pool range definition. For example: 192.0.2.1 - 192.0.2.10 becomes
// 192.0.2.1-192.0.2.10. If the pool is specified using the prefix form, it converts
// it to the range form.
func (p *PoolKnownParameters) UnmarshalJSON(data []byte) error {
	type t PoolKnownParameters
	if err := json.Unmarshal(data, (*t)(p)); err != nil {
		return err
	}
	if strings.ContainsRune(p.Pool, '-') {
		buf := bytes.Buffer{}
		for i := 0; i < len(p.Pool); i++ {
			if !unicode.IsSpace(rune(p.Pool[i])) {
				buf.WriteByte(p.Pool[i])
			}
		}
		p.Pool = buf.String()
		return nil
	}
	lb, ub, err := storkutil.ParseIPRange(p.Pool)
	if err != nil {
		return errors.Errorf("invalid pool prefix %s", p.Pool)
	}
	p.Pool = fmt.Sprintf("%s-%s", lb, ub)
	return nil
}

// Unmarshals the JSON data into the Pool structure. The output contains
// the known parameters and a map of unknown parameters.
func (p *Pool) UnmarshalJSON(data []byte) error {
	poolWithUnknown := WithUnknown[PoolKnownParameters]{}
	if err := json.Unmarshal(data, &poolWithUnknown); err != nil {
		return err
	}
	*p = Pool{
		PoolKnownParameters: poolWithUnknown.Known,
		UnknownParameters:   poolWithUnknown.Unknown,
	}
	return nil
}

// Marshals the Pool structure into JSON. The output contains the known
// parameters and a map of unknown parameters.
func (p Pool) MarshalJSON() ([]byte, error) {
	poolWithUnknown := WithUnknown[PoolKnownParameters]{
		Known:   p.PoolKnownParameters,
		Unknown: p.UnknownParameters,
	}
	return json.Marshal(poolWithUnknown)
}

// Returns the pool boundaries (lower, upper bounds).
func (p Pool) GetBoundaries() (net.IP, net.IP, error) {
	lb, ub, err := storkutil.ParseIPRange(p.Pool)
	return lb, ub, err
}

// Returns a pointer to the pool parameters.
func (p Pool) GetPoolParameters() *PoolParameters {
	return &PoolParameters{
		ClientClassParameters: p.ClientClassParameters,
		PoolID:                p.PoolID,
		UnknownParameters:     p.UnknownParameters,
	}
}
