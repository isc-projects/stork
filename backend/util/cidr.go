package storkutil

import (
	"bytes"
	"fmt"
	"math/big"
	"net"
	"strings"

	cidr "github.com/apparentlymart/go-cidr/cidr"
	"github.com/pkg/errors"
)

// IP protocol type.
type IPType int

// IP protocol type enum.
const (
	IPv4 IPType = 4
	IPv6 IPType = 6
)

// Structure returned by ParseIP function. It comprises the information about
// the parsed IP address or delegated prefix.
type ParsedIP struct {
	NetworkAddress string // Full address or delegated prefix e.g. 192.0.2.0/24.
	Protocol       IPType // Detected IP type: IPv4 or IPv6.
	NetworkPrefix  string // Prefix part (slash and length excluded).
	PrefixLength   int    // Network address mask.
	Prefix         bool   // Boolean indicating if it is an address or prefix.
	CIDR           bool   // Boolean indicating if parsed value was in CIDR form.
	IP             net.IP
	IPNet          *net.IPNet
}

// Turns IP address into CIDR. If the IP address already seems to be using
// CIDR notation, it is returned.
func MakeCIDR(address string) (string, error) {
	if !strings.Contains(address, "/") {
		ip := net.ParseIP(address)
		if ip == nil {
			return address, errors.Errorf("provided string %s is not a valid IP address", address)
		}
		ip4 := ip.To4()
		if ip4 != nil {
			address += "/32"
		} else {
			address += "/128"
		}
	}
	return address, nil
}

// Parses an IP address or prefix and returns parsed information in the
// structure. If the specified value is invalid, a nil structure is
// returned.
func ParseIP(address string) *ParsedIP {
	parsed := &ParsedIP{}

	// Check if this is an IP address without a prefix length.
	parsed.IP = net.ParseIP(address)
	if parsed.IP == nil {
		// Apparently it comprises a prefix length.
		ip, net, err := net.ParseCIDR(address)
		if err != nil {
			// It is neither an IP address nor prefix.
			return nil
		}
		// Gather prefix information from the network.
		parsed.IP = ip
		parsed.IPNet = net
	}

	if parsed.IPNet != nil {
		// Check prefix length.
		ones, bits := parsed.IPNet.Mask.Size()
		if ones != bits {
			// This seems to be a prefix.
			parsed.NetworkAddress = parsed.IPNet.String()
			parsed.NetworkPrefix = parsed.IPNet.IP.String()
			parsed.PrefixLength = ones
			parsed.Prefix = true
		}
		// Caller specified it with a prefix length.
		parsed.CIDR = true
	}

	// Prefix length wasn't specified, so we need to parse the
	// IP address.
	if !parsed.Prefix {
		parsed.NetworkAddress = parsed.IP.String()
		parsed.NetworkPrefix = parsed.IP.String()
	}

	// Check if this is IPv4 or IPv6 address/prefix.
	if parsed.IP.To4() != nil {
		if parsed.PrefixLength == 0 {
			parsed.PrefixLength = 32
		}
		parsed.Protocol = IPv4
	} else {
		if parsed.PrefixLength == 0 {
			parsed.PrefixLength = 128
		}
		parsed.Protocol = IPv6
	}
	// Parsing finished.
	return parsed
}

// Returns lower and upper bound addresses of the address range. The address
// range may follow two conventions, e.g., 192.0.2.1 - 192.0.3.10
// or 192.0.2.0/24. Both IPv4 and IPv6 ranges are supported by this function.
func ParseIPRange(ipRange string) (net.IP, net.IP, error) {
	// Let's try to see if the range is specified as a pair of upper
	// and lower bound addresses.
	s := strings.Split(ipRange, "-")
	for i := 0; i < len(s); i++ {
		s[i] = strings.TrimSpace(s[i])
	}
	// The length of 2 means that the two addresses with hyphen were specified.
	switch len(s) {
	case 2:
		ips := []net.IP{}
		families := []int{}
		for _, ipStr := range s {
			// Check if the specified value is even an IP address.
			ip := net.ParseIP(ipStr)
			if ip == nil {
				// It is not an IP address. Bail...
				err := errors.Errorf("unable to parse the IP address %s", ipStr)
				return nil, nil, err
			}
			// It is an IP address, so let's see if it converts to IPv4 or IPv6.
			// In both cases, remember the family.
			if ip.To4() != nil {
				families = append(families, 4)
				ips = append(ips, ip)
			} else {
				families = append(families, 6)
				ips = append(ips, ip)
			}
			// If we already checked both addresses, let's compare their families.
			if (len(families) > 1) && (families[0] != families[1]) {
				// IPv4 and IPv6 address given. This is unacceptable.
				err := errors.Errorf("IP addresses in the IP range %s must belong to the same family",
					ipRange)
				return nil, nil, err
			}
		}
		return ips[0], ips[1], nil

	case 1:
		// There is one token only, so apparently this is a range provided as a prefix.
		_, net, err := net.ParseCIDR(s[0])
		if err != nil {
			err = errors.Errorf("unable to parse the pool prefix %s", s[0])
			return nil, nil, err
		}
		// For this prefix find an upper and lower bound address.
		lb, ub := cidr.AddressRange(net)
		return lb, ub, nil

	default:
		// No other formats for the address range are accepted.
		err := errors.Errorf("unable to parse the IP range %s", ipRange)
		return nil, nil, err
	}
}

// Checks if an IP address is within the range of addresses between the
// lb (lower bound) and ub (upper bound).
func (parsed *ParsedIP) IsInRange(lb, ub net.IP) bool {
	// Require that it is an IPv4 or IPv6 address. Delegated prefix should
	// be verified by IsInPrefixRange.
	if !parsed.Prefix {
		ip4 := parsed.IP
		// Convert to 16 bytes. It makes the comparison common for both
		// the IPv4 and IPv6 case.
		ip16 := ip4.To16()
		if bytes.Compare(ip16, lb) >= 0 && bytes.Compare(ip16, ub) <= 0 {
			return true
		}
	}
	// Out of range.
	return false
}

// Checks if a prefix is within the range of delegated prefixes specified
// by the prefix, prefix length and delegated length.
func (parsed *ParsedIP) IsInPrefixRange(prefix string, prefixLen, delegatedLen int) bool {
	// Require that it is a prefix. IP addresses should be verified using
	// the IsInRange function. Also, make sure that the container prefix
	// length is lower than the tested delegated prefix. Finally, the
	// tested delegated prefix length must be equal to the containing
	// delegated prefix length.
	if parsed.Prefix && prefixLen <= parsed.PrefixLength && delegatedLen == parsed.PrefixLength {
		// Combine the prefix and the length in the IPNet.
		_, rangeNetwork, err := net.ParseCIDR(FormatCIDRNotation(prefix, prefixLen))
		if err != nil {
			return false
		}
		// Mask the tested IP using the containing prefix mask. They
		// must be equal.
		parsedIPMasked := parsed.IP.Mask(rangeNetwork.Mask)
		if parsedIPMasked.Equal(rangeNetwork.IP) {
			return true
		}
	}
	return false
}

// Calculates the number of addresses in the address range.
func CalculateRangeSize(lb, ub net.IP) *big.Int {
	size := big.NewInt(0)
	size.Add(size, big.NewInt(0).SetBytes(ub))
	size.Sub(size, big.NewInt(0).SetBytes(lb))
	size.Add(size, big.NewInt(1))
	return size
}

// Calculates the number of delegated prefixes in the delegated prefix range.
func CalculateDelegatedPrefixRangeSize(prefixLength, delegatedLength int) *big.Int {
	if delegatedLength < prefixLength {
		// Invalid arguments.
		return big.NewInt(0)
	}

	// Number of delegated prefixes = 2 ^ (delegated length - prefix length).
	return big.NewInt(0).Exp(
		big.NewInt(2),
		big.NewInt(int64(delegatedLength-prefixLength)),
		nil,
	)
}

// Returns network prefix as a binary string without delimiters. It
// contains only the prefix bytes without leading zeros. The IPv4 prefixes are
// prepended by the constant term to avoid collisions with the IPv6 ones.
// If the network prefix is invalid, the empty string is returned.
func (parsed *ParsedIP) GetNetworkPrefixAsBinary() string {
	ip := net.ParseIP(parsed.NetworkPrefix)
	if ip == nil {
		return ""
	}

	ip = ip.To16()
	prefixLength := parsed.PrefixLength
	if prefixLength <= 0 {
		return ""
	}

	if parsed.Protocol == IPv4 {
		prefixLength += 12 * 8
	}

	parts := make([]string, len(ip))
	for i, tn := range ip {
		parts[i] = fmt.Sprintf("%08b", tn)
	}
	prefixBin := strings.Join(parts, "")
	if len(prefixBin) < prefixLength {
		// Invalid prefix length
		return ""
	}
	return prefixBin[0:prefixLength]
}

// Returns prefix with a length in form: prefix/length.
func (parsed *ParsedIP) GetNetworkPrefixWithLength() string {
	return FormatCIDRNotation(parsed.NetworkPrefix, parsed.PrefixLength)
}

// Combines the IP and mask to a single string using
// the [IP]/[MASK] notation.
func FormatCIDRNotation(ip string, mask int) string {
	return fmt.Sprintf("%s/%d", ip, mask)
}
