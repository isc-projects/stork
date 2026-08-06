package keadata

import (
	"encoding/json"

	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"

	log "github.com/sirupsen/logrus"
)

// Constants representing various lease states in Kea.  Other states can be
// added in the future in Kea. In such case this constants list should be
// updated to include the new states.
const (
	// A valid (non-expired) lease.
	LeaseStateDefault uint32 = 0
	// A lease where a client sent a decline message because it detected another client using the address already.
	LeaseStateDeclined = 1
	// A lease where the valid lifetime has elapsed, but which is retained so that if the same client returns, they can get the same address.
	LeaseStateExpiredReclaimed uint32 = 2
	// A lease where a client sent a release message, but which is retained so that if they ask again, they can get the same address.
	LeaseStateReleased uint32 = 3
	// A lease where the client made up their own IP address and has notified the DHCP server which address they picked. (Only supported by DHCPv6.)
	LeaseStateRegistered uint32 = 4
)

// Represents a DHCP lease fetched from Kea.
type Lease struct {
	Family          storkutil.IPType `json:"-"`
	ClientID        *ColonSepHexStr  `json:"client-id,omitempty"`
	Hostname        string           `json:"hostname,omitempty"`
	HWAddress       string           `json:"hw-address,omitempty"`
	HWAddressSource string           `json:"hw-address-source,omitempty"`
	// HWType is a uint16 in Kea, but the smallest unsigned integer type which Protobuf
	// supports is uint32, so rather than waste a bunch of CPU cycles converting it
	// back and forth half a dozen times, this expands it once and leaves it that way.
	HWType            *uint32         `json:"hw-type,omitempty"`
	DUID              *ColonSepHexStr `json:"duid,omitempty"`
	IPAddress         string          `json:"ip-address,omitempty"`
	Type              string          `json:"type,omitempty"`
	CLTT              uint64          `json:"cltt,omitempty"`
	State             uint32          `json:"state,omitempty" pg:",use_zero"`
	UserContext       map[string]any  `json:"user-context,omitempty"`
	ValidLifetime     uint32          `json:"valid-lft,omitempty"`
	IAID              uint32          `json:"iaid,omitempty"`
	PreferredLifetime uint32          `json:"preferred-lft,omitempty"`
	LocalSubnetID     uint32          `json:"subnet-id,omitempty"`
	FqdnFwd           bool            `json:"fqdn-fwd,omitempty"`
	FqdnRev           bool            `json:"fqdn-rev,omitempty"`
	PrefixLength      uint8           `json:"prefix-len,omitempty"`
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv4 lease.
func NewLease4(
	ip,
	hwAddress,
	clientID string,
	cltt uint64,
	validLifetime,
	subnetID uint32,
	fqdnFwd,
	fqdnRev bool,
	hostname string,
	state uint32,
	userCtx map[string]any,
) Lease {
	return Lease{
		Family:        storkutil.IPv4,
		IPAddress:     ip,
		HWAddress:     hwAddress,
		CLTT:          cltt,
		ValidLifetime: validLifetime,
		LocalSubnetID: subnetID,
		State:         state,
		ClientID:      NewColonSepHexStr(&clientID),
		FqdnFwd:       fqdnFwd,
		FqdnRev:       fqdnRev,
		Hostname:      hostname,
		UserContext:   userCtx,
	}
}

// Create a new Lease, filling in all the fields which are appropriate for a
// DHCPv6 lease.
func NewLease6(
	ip,
	duid,
	leaseType string,
	cltt uint64,
	validLifetime,
	subnetID,
	prefLifetime,
	iaid uint32,
	prefixLen uint8,
	fqdnFwd,
	fqdnRev bool,
	hostname,
	hwaddr string,
	state uint32,
	userCtx map[string]any,
	hwtype *uint32,
	hwaddrSource string,
) Lease {
	return Lease{
		Type:              leaseType,
		Family:            storkutil.IPv6,
		IPAddress:         ip,
		DUID:              NewColonSepHexStr(&duid),
		CLTT:              cltt,
		ValidLifetime:     validLifetime,
		LocalSubnetID:     subnetID,
		State:             state,
		PrefixLength:      prefixLen,
		PreferredLifetime: prefLifetime,
		IAID:              iaid,
		FqdnFwd:           fqdnFwd,
		FqdnRev:           fqdnRev,
		Hostname:          hostname,
		HWAddress:         hwaddr,
		HWAddressSource:   hwaddrSource,
		UserContext:       userCtx,
		HWType:            hwtype,
	}
}

// Convert the Lease into the Lease Protobuf structure returned by the agent's
// gRPC API.
func (lease *Lease) ToGRPC() agentapi.Lease {
	userCtxByte, err := json.Marshal(lease.UserContext)
	var userCtxStr string
	if err != nil {
		log.WithError(err).Debug("failed to serialize JSON user context")
	} else {
		userCtxStr = string(userCtxByte)
	}
	return agentapi.Lease{
		Type:            lease.Type,
		Family:          agentapi.Lease_IPAddrFamily(lease.Family), // #nosec: G115
		IpAddress:       lease.IPAddress,
		HwAddress:       lease.HWAddress,
		Duid:            lease.DUID.String(),
		Cltt:            lease.CLTT,
		ValidLifetime:   uint64(lease.ValidLifetime),
		SubnetID:        lease.LocalSubnetID,
		State:           lease.State,
		PrefixLen:       uint32(lease.PrefixLength),
		ClientID:        lease.ClientID.String(),
		FqdnFwd:         lease.FqdnFwd,
		FqdnRev:         lease.FqdnRev,
		Iaid:            lease.IAID,
		Hostname:        lease.Hostname,
		HwAddressSource: lease.HWAddressSource,
		UserContext:     userCtxStr,
		HwType:          lease.HWType,
	}
}
