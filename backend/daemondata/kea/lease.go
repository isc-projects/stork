package keadata

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
)

// Represents a DHCP lease fetched from Kea.
type Lease struct {
	ClientID          string         `json:"client-id,omitempty"`
	CLTT              uint64         `json:"cltt,omitempty"`
	DUID              string         `json:"duid,omitempty"`
	FqdnFwd           bool           `json:"fqdn-fwd,omitempty"`
	FqdnRev           bool           `json:"fqdn-rev,omitempty"`
	Hostname          string         `json:"hostname,omitempty"`
	HWAddress         string         `json:"hw-address,omitempty"`
	IAID              uint32         `json:"iaid,omitempty"`
	IPAddress         string         `json:"ip-address,omitempty"`
	PreferredLifetime uint32         `json:"preferred-lft,omitempty"`
	PrefixLength      uint8          `json:"prefix-len,omitempty"`
	State             int            `json:"state,omitempty"`
	SubnetID          uint32         `json:"subnet-id,omitempty"`
	Type              string         `json:"type,omitempty"`
	UserContext       map[string]any `json:"user-context,omitempty"`
	ValidLifetime     uint32         `json:"valid-lft,omitempty"`
}
