package dbmodel

import (
	"net"
	"strings"
	"time"

	cidr "github.com/apparentlymart/go-cidr/cidr"
	errors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Reflects IPv4 or IPv6 address pool.
type AddressPool struct {
	ID         int64
	CreatedAt  time.Time
	LowerBound string
	UpperBound string
	SubnetID   int64
	Subnet     *Subnet
}

// Reflects IPv6 address pool.
type PrefixPool struct {
	ID           int64
	CreatedAt    time.Time
	Prefix       string
	DelegatedLen int
	SubnetID     int64
	Subnet       *Subnet
}

// Creates new instance of the address pool from the address range. The
// address range may follow two conventions, e.g. 192.0.2.1 - 192.0.3.10
// or 192.0.2.0/24. Both IPv4 and IPv6 pools are supported by this function.
func NewAddressPoolFromRange(addressRange string) (*AddressPool, error) {
	// Let's try to see if the range is specified as a pair of upper
	// and lower bound addresses.
	s := strings.Split(addressRange, "-")
	for i := 0; i < len(s); i++ {
		s[i] = strings.TrimSpace(s[i])
	}
	pool := &AddressPool{}

	// The length of 2 means that the two addresses with hyphen were specified.
	switch len(s) {
	case 2:
		families := []int{}
		for _, ipStr := range s {
			// Check if the specified value is even an IP address.
			ip := net.ParseIP(ipStr)
			if ip == nil {
				// It is not an IP address. Bail...
				err := errors.Errorf("unable to parse the IP address %s", ipStr)
				return nil, err
			}
			// It is an IP address, so let's see if it converts to IPv4 or IPv6.
			// In both cases, remember the family.
			if ip.To4() != nil {
				families = append(families, 4)
			} else {
				families = append(families, 6)
			}
			// If we already checked both addresses, let's compare their families.
			if (len(families) > 1) && (families[0] != families[1]) {
				// IPv4 and IPv6 address given. This is unacceptable.
				err := errors.Errorf("IP addresses in the pool range %s must belong to the same family",
					addressRange)
				return nil, err
			}
		}
		// Everything was good, so let's put the addresses into the pool instance.
		// ToLower ensures that the IPv6 address digits are converted to lower case.
		pool.LowerBound = strings.ToLower(s[0])
		pool.UpperBound = strings.ToLower(s[1])

	case 1:
		// There is one token only, so apparently this is a range provided as a prefix.
		_, net, err := net.ParseCIDR(s[0])
		if err != nil {
			err := errors.Errorf("unable to parse the pool prefix %s", s[0])
			return nil, err
		}
		// For this prefix find an upper and lower bound address.
		rb, re := cidr.AddressRange(net)
		pool.LowerBound = rb.String()
		pool.UpperBound = re.String()

	default:
		// No other formats for the address range are accepted.
		err := errors.Errorf("unable to parse the pool range %s", addressRange)
		return nil, err
	}
	// We have the pool.
	return pool, nil
}

// Creates new instance of the pool for prefix delegation from the prefix
// and delegated length. It validates the prefix provided to verify if it
// follows CIDR notation.
func NewPrefixPool(prefix string, delegatedLen int) (*PrefixPool, error) {
	ipAddr, net, err := net.ParseCIDR(prefix)
	if err != nil {
		err = errors.Errorf("unable to parse the pool prefix %s", prefix)
		return nil, err
	}
	// This prefix must not convert to IPv4. Only IPv6 is allowed.
	if ipAddr.To4() != nil {
		err = errors.Errorf("specified prefix %s is not an IPv6 prefix", prefix)
		return nil, err
	}
	pool := &PrefixPool{}
	pool.Prefix = net.String()
	pool.DelegatedLen = delegatedLen
	return pool, nil
}

// Adds address pool to the database.
func AddAddressPool(db *dbops.PgDB, pool *AddressPool) error {
	if pool.SubnetID == 0 && pool.Subnet == nil {
		err := errors.Errorf("subnet must be specified while adding new pool %s-%s into the database",
			pool.LowerBound, pool.UpperBound)
		return err
	}

	// In case, the caller specified pointer to the subnet rather than subnet id
	// we have to set the subnet id on our own.
	if pool.SubnetID == 0 && pool.Subnet != nil {
		pool.SubnetID = pool.Subnet.ID
	}

	_, err := db.Model(pool).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new address pool %s-%s into the database for subnet %d",
			pool.LowerBound, pool.UpperBound, pool.SubnetID)
	}
	return err
}

// Adds prefix pool to the database.
func AddPrefixPool(db *dbops.PgDB, pool *PrefixPool) error {
	if pool.Subnet.ID == 0 && pool.Subnet == nil {
		err := errors.Errorf("subnet must be specified while adding new prefix pool %s into the database",
			pool.Prefix)
		return err
	}

	// In case, the caller specified pointer to the subnet rather than subnet id
	// we have to set the subnet id on our own.
	if pool.SubnetID == 0 && pool.Subnet != nil {
		pool.SubnetID = pool.Subnet.ID
	}

	_, err := db.Model(pool).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new prefix pool %s into the database for subnet %d",
			pool.Prefix, pool.SubnetID)
	}
	return err
}

// Deletes IPv4 or IPv6 address pool from the database.
func DeleteAddressPool(db *dbops.PgDB, poolID int64) error {
	pool := &AddressPool{
		ID: poolID,
	}
	_, err := db.Model(pool).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting the address pool with id %d", poolID)
	}
	return err
}

// Deletes IPv6 address pool from the database.
func DeletePrefixPool(db *dbops.PgDB, poolID int64) error {
	pool := &PrefixPool{
		ID: poolID,
	}
	_, err := db.Model(pool).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting the prefix pool with id %d", poolID)
	}
	return err
}
