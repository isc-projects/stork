package keadata

import (
	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

// Constants representing various lease states in Kea. A valid
// (non-expired) lease is in the default state. A lease for
// which a client detected that it is used by another client
// and sent the DHCP decline message is in the declined state.
// A lease for which valid lifetime elapsed and the Kea server
// detected that the lease is expired can be removed from the
// database or left in the expired-reclaimed state. Keeping the
// lease in the expired-reclaimed state increases chances that
// the returning client will be allocated the same lease.
// However, such lease can be allocated to any client requesting
// it. Other states can be added in the future in Kea. In such
// case this constants list should be extended.
const (
	LeaseStateDefault          = 0
	LeaseStateDeclined         = 1
	LeaseStateExpiredReclaimed = 2
	LeaseStateReleased         = 3
)

// Represents a DHCP lease fetched from Kea.
type Lease struct {
	IPVersion         storkutil.IPType `json:"-"`
	ClientID          string           `json:"client-id,omitempty"`
	Hostname          string           `json:"hostname,omitempty"`
	HWAddress         string           `json:"hw-address,omitempty"`
	DUID              string           `json:"duid,omitempty"`
	IPAddress         string           `json:"ip-address,omitempty"`
	Type              string           `json:"type,omitempty"`
	CLTT              uint64           `json:"cltt,omitempty"`
	State             int              `json:"state,omitempty"`
	UserContext       map[string]any   `json:"user-context,omitempty"`
	ValidLifetime     uint32           `json:"valid-lft,omitempty"`
	IAID              uint32           `json:"iaid,omitempty"`
	PreferredLifetime uint32           `json:"preferred-lft,omitempty"`
	SubnetID          uint32           `json:"subnet-id,omitempty"`
	FqdnFwd           bool             `json:"fqdn-fwd,omitempty"`
	FqdnRev           bool             `json:"fqdn-rev,omitempty"`
	PrefixLength      uint8            `json:"prefix-len,omitempty"`
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv4 lease.
func NewLease4(ip string, hwAddress string, cltt uint64, validLifetime uint32, subnetID uint32, state int) Lease {
	return Lease{
		IPVersion:     storkutil.IPv4,
		IPAddress:     ip,
		HWAddress:     hwAddress,
		CLTT:          cltt,
		ValidLifetime: validLifetime,
		SubnetID:      subnetID,
		State:         state,
	}
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv6 lease.
func NewLease6(ip string, duid string, cltt uint64, validLifetime uint32, subnetID uint32, state int, prefixLen uint32) Lease {
	return Lease{
		IPVersion:     storkutil.IPv6,
		IPAddress:     ip,
		DUID:          duid,
		CLTT:          cltt,
		ValidLifetime: validLifetime,
		SubnetID:      subnetID,
		State:         state,
		PrefixLength:  uint8(prefixLen),
	}
}

// Convert the Lease into the Lease Protobuf structure returned by the agent's
// gRPC API.
func (lease *Lease) ToGRPC() agentapi.Lease {
	return agentapi.Lease{
		IpVersion:     agentapi.Lease_IPVersion(lease.IPVersion),
		IpAddress:     lease.IPAddress,
		HwAddress:     lease.HWAddress,
		Duid:          lease.DUID,
		Cltt:          lease.CLTT,
		ValidLifetime: uint64(lease.ValidLifetime),
		SubnetID:      lease.SubnetID,
		State:         uint32(lease.State),
		PrefixLen:     uint32(lease.PrefixLength),
	}
}
