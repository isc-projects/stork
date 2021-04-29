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
	ClientID          string `mapstructure:"client-id" json:"client-id,omitempty"`
	CLTT              uint64 `mapstructure:"cltt" json:"cltt,omitempty"`
	DUID              string `mapstructure:"duid" json:"duid,omitempty"`
	FqdnFwd           bool   `mapstructure:"fqdn-fwd" json:"fqdn-fwd,omitempty"`
	FqdnRev           bool   `mapstructure:"fqdn-rev" json:"fqdn-rev,omitempty"`
	Hostname          string `mapstructure:"hostname" json:"hostname,omitempty"`
	HWAddress         string `mapstructure:"hw-address" json:"hw-address,omitempty"`
	IAID              uint32 `mapstructure:"iaid" json:"iaid,omitempty"`
	IPAddress         string `mapstructure:"ip-address" json:"ip-address,omitempty"`
	PreferredLifetime uint32 `mapstructure:"preferred-lft" json:"preferred-lft,omitempty"`
	PrefixLength      uint8  `mapstructure:"prefix-len" json:"prefix-len,omitempty"`
	State             int    `mapstructure:"state" json:"state,omitempty"`
	SubnetID          uint32 `mapstructure:"subnet-id" json:"subnet-id,omitempty"`
	Type              string `mapstructure:"type" json:"type,omitempty"`
	UserContext       string `mapstructure:"user-context" json:"user-context,omitempty"`
	ValidLifetime     uint32 `mapstructure:"valid-lft" json:"valid-lft,omitempty"`
}
