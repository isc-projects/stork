package daemonname

// Defines the consistent names of the daemons we support. They are intended
// to be used throughout the codebase. This package should not import any
// packages from the Stork repository to avoid possible circular dependencies.
type Name string

const (
	Bind9   Name = "named"
	DHCPv4  Name = "dhcp4"
	DHCPv6  Name = "dhcp6"
	NetConf Name = "netconf"
	D2      Name = "d2"
	CA      Name = "ca"
	PDNS    Name = "pdns"
)

// Indicates if the daemon name is a Kea daemon name.
func (dn Name) IsKea() bool {
	switch dn {
	case DHCPv4, DHCPv6, D2, CA:
		return true
	default:
		return false
	}
}

// Indicates if the daemon name is a DHCP daemon name.
func (dn Name) IsDHCP() bool {
	switch dn {
	case DHCPv4, DHCPv6:
		return true
	default:
		return false
	}
}

// Indicates if the daemon name is a DNS daemon name.
func (dn Name) IsDNS() bool {
	switch dn {
	case Bind9, PDNS:
		return true
	default:
		return false
	}
}

// Parses the daemon name from string. It returns false if the
// daemon name is not recognized.
func Parse(name string) (Name, bool) {
	switch name {
	case string(Bind9):
		return Bind9, true
	case string(DHCPv4):
		return DHCPv4, true
	case string(DHCPv6):
		return DHCPv6, true
	case string(NetConf):
		return NetConf, true
	case string(D2):
		return D2, true
	case string(CA):
		return CA, true
	case string(PDNS):
		return PDNS, true
	default:
		return Name(""), false
	}
}
