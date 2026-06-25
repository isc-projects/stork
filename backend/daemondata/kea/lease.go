package keadata

import (
	"encoding/json"

	"github.com/pkg/errors"
	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

// Kea stores the state in uint32, but it casts it to int when returning it
// over the REST API. This state provides a custom JSON unmarshaller to handle
// this conversion.
type LeaseState uint32

// Unmarshals the lease state as int and then converts it to uint32.
// The possible overflow is intended, however, since the range of state's
// values is very small, it should never happen in practice.
func (s *LeaseState) UnmarshalJSON(data []byte) error {
	var stateInt int64
	err := json.Unmarshal(data, &stateInt)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal lease state from JSON")
		return err
	}
	*s = LeaseState(stateInt) //nolint:gosec
	return nil
}

// Marshalls the lease state as Kea does, i.e. it is converted to int with
// possible overflow and then marshalled to JSON.
func (s LeaseState) MarshalJSON() ([]byte, error) {
	stateInt := int(s)
	data, err := json.Marshal(stateInt)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal lease state to JSON")
		return nil, err
	}
	return data, nil
}

// Constants representing various lease states in Kea.  Other states can be
// added in the future in Kea. In such case this constants list should be
// updated to include the new states.
const (
	// A valid (non-expired) lease.
	LeaseStateDefault LeaseState = 0
	// A lease where a client sent a decline message because it detected another client using the address already.
	LeaseStateDeclined LeaseState = 1
	// A lease where the valid lifetime has elapsed, but which is retained so that if the same client returns, they can get the same address.
	LeaseStateExpiredReclaimed LeaseState = 2
	// A lease where a client sent a release message, but which is retained so that if they ask again, they can get the same address.
	LeaseStateReleased LeaseState = 3
	// A lease where the client made up their own IP address and has notified the DHCP server which address they picked. (Only supported by DHCPv6.)
	LeaseStateRegistered LeaseState = 4
)

// Represents a DHCP lease fetched from Kea.
type Lease struct {
	Family    storkutil.IPType `json:"-"`
	ClientID  *ColonSepHexStr  `json:"client-id,omitempty"`
	Hostname  string           `json:"hostname,omitempty"`
	HWAddress string           `json:"hw-address,omitempty"`
	DUID      *ColonSepHexStr  `json:"duid,omitempty"`
	IPAddress string           `json:"ip-address,omitempty"`
	Type      string           `json:"type,omitempty"`
	CLTT      uint64           `json:"cltt,omitempty"`
	// Kea stores the state in uint32, but it casts it to int when returning it
	// over the REST API.
	State             LeaseState     `json:"state,omitempty" pg:",use_zero"`
	UserContext       map[string]any `json:"user-context,omitempty"`
	ValidLifetime     uint32         `json:"valid-lft,omitempty"`
	IAID              uint32         `json:"iaid,omitempty"`
	PreferredLifetime uint32         `json:"preferred-lft,omitempty"`
	LocalSubnetID     uint32         `json:"subnet-id,omitempty"`
	FqdnFwd           bool           `json:"fqdn-fwd,omitempty"`
	FqdnRev           bool           `json:"fqdn-rev,omitempty"`
	PrefixLength      uint8          `json:"prefix-len,omitempty"`
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv4 lease.
func NewLease4(ip, hwAddress, clientID string, cltt uint64, validLifetime, subnetID uint32, state LeaseState) Lease {
	return Lease{
		Family:        storkutil.IPv4,
		IPAddress:     ip,
		HWAddress:     hwAddress,
		CLTT:          cltt,
		ValidLifetime: validLifetime,
		LocalSubnetID: subnetID,
		State:         state,
		ClientID:      NewColonSepHexStr(&clientID),
	}
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv6 lease.
func NewLease6(ip, duid string, cltt uint64, validLifetime, subnetID uint32, state LeaseState, prefixLen uint8) Lease {
	return Lease{
		Family:        storkutil.IPv6,
		IPAddress:     ip,
		DUID:          NewColonSepHexStr(&duid),
		CLTT:          cltt,
		ValidLifetime: validLifetime,
		LocalSubnetID: subnetID,
		State:         state,
		PrefixLength:  prefixLen,
	}
}

// Convert the Lease into the Lease Protobuf structure returned by the agent's
// gRPC API.
func (lease *Lease) ToGRPC() agentapi.Lease {
	return agentapi.Lease{
		Family:        agentapi.Lease_IPAddrFamily(lease.Family), //nolint:gosec
		IpAddress:     lease.IPAddress,
		HwAddress:     lease.HWAddress,
		Duid:          lease.DUID.String(),
		Cltt:          lease.CLTT,
		ValidLifetime: uint64(lease.ValidLifetime),
		SubnetID:      lease.LocalSubnetID,
		State:         uint32(lease.State),
		PrefixLen:     uint32(lease.PrefixLength),
		ClientID:      lease.ClientID.String(),
	}
}
