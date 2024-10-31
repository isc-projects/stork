package dbmodel

import (
	"net"
	"time"

	errors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Interface checks.
var (
	_ keaconfig.AddressPool        = (*AddressPool)(nil)
	_ dhcpmodel.PrefixPoolAccessor = (*PrefixPool)(nil)
)

// Reflects IPv4 or IPv6 address pool.
type AddressPool struct {
	DHCPOptionSet

	ID            int64
	CreatedAt     time.Time
	LowerBound    string
	UpperBound    string
	LocalSubnetID int64
	LocalSubnet   *LocalSubnet `pg:"rel:has-one"`

	KeaParameters *keaconfig.PoolParameters
}

// Returns lower pool boundary.
func (ap *AddressPool) GetLowerBound() string {
	return ap.LowerBound
}

// Returns upper pool boundary.
func (ap *AddressPool) GetUpperBound() string {
	return ap.UpperBound
}

// Returns a slice of interfaces describing the DHCP options for a pool.
func (ap *AddressPool) GetKeaParameters() *keaconfig.PoolParameters {
	return ap.KeaParameters
}

// Returns a slice of interfaces describing the DHCP options for a pool.
func (ap *AddressPool) GetDHCPOptions() (accessors []dhcpmodel.DHCPOptionAccessor) {
	for i := range ap.DHCPOptionSet.Options {
		accessors = append(accessors, ap.DHCPOptionSet.Options[i])
	}
	return
}

// Checks equality of the address pool's own data without database-related members
// and references.
func (ap *AddressPool) HasEqualData(other *AddressPool) bool {
	return ap.LowerBound == other.LowerBound &&
		ap.UpperBound == other.UpperBound
}

// Reflects IPv6 address pool.
type PrefixPool struct {
	ID                int64
	CreatedAt         time.Time
	Prefix            string
	DelegatedLen      int
	ExcludedPrefix    string
	DHCPOptionSet     []DHCPOption
	DHCPOptionSetHash string
	LocalSubnetID     int64
	LocalSubnet       *LocalSubnet `pg:"rel:has-one"`

	KeaParameters *keaconfig.PoolParameters
}

// Returns a pointer to a structure holding the delegated prefix data.
func (pp *PrefixPool) GetModel() *dhcpmodel.PrefixPool {
	return &dhcpmodel.PrefixPool{
		Prefix:         pp.Prefix,
		DelegatedLen:   pp.DelegatedLen,
		ExcludedPrefix: pp.ExcludedPrefix,
	}
}

// Returns a slice of interfaces describing the DHCP options for a prefix pool.
func (pp *PrefixPool) GetKeaParameters() *keaconfig.PoolParameters {
	return pp.KeaParameters
}

// Returns a slice of interfaces describing the DHCP options for a pool.
func (pp *PrefixPool) GetDHCPOptions() (accessors []dhcpmodel.DHCPOptionAccessor) {
	for i := range pp.DHCPOptionSet {
		accessors = append(accessors, pp.DHCPOptionSet[i])
	}
	return
}

// Checks equality of the prefix pool's own data without database-related members
// and references.
func (pp *PrefixPool) HasEqualData(other *PrefixPool) bool {
	return pp.Prefix == other.Prefix &&
		pp.DelegatedLen == other.DelegatedLen &&
		pp.ExcludedPrefix == other.ExcludedPrefix
}

// Creates a new address pool given the address range.
func NewAddressPool(lb, ub net.IP) *AddressPool {
	pool := &AddressPool{
		LowerBound: lb.String(),
		UpperBound: ub.String(),
	}
	return pool
}

// Creates new instance of the address pool from the address range. The
// address range may follow two conventions, e.g. 192.0.2.1 - 192.0.3.10
// or 192.0.2.0/24. Both IPv4 and IPv6 pools are supported by this function.
func NewAddressPoolFromRange(addressRange string) (*AddressPool, error) {
	lb, ub, err := storkutil.ParseIPRange(addressRange)
	if err == nil {
		pool := &AddressPool{
			LowerBound: lb.String(),
			UpperBound: ub.String(),
		}
		return pool, nil
	}
	return nil, err
}

// Creates new instance of the pool for prefix delegation from the prefix,
// delegated length, and optional excluded prefix. It validates the prefix
// provided to verify if it follows CIDR notation.
func NewPrefixPool(prefix string, delegatedLen int, excludedPrefix string) (*PrefixPool, error) {
	prefixIP, prefixNet, err := net.ParseCIDR(prefix)
	if err != nil {
		err = errors.Errorf("unable to parse the pool prefix %s", prefix)
		return nil, err
	}
	// This prefix must not convert to IPv4. Only IPv6 is allowed.
	if prefixIP.To4() != nil {
		err = errors.Errorf("specified prefix %s is not an IPv6 prefix", prefix)
		return nil, err
	}

	pool := &PrefixPool{
		Prefix:       prefixNet.String(),
		DelegatedLen: delegatedLen,
	}

	if excludedPrefix != "" {
		excludedIP, excludedNet, err := net.ParseCIDR(excludedPrefix)
		if err != nil {
			return nil, errors.Errorf("unable to parse the excluded prefix %s", excludedPrefix)
		}

		if excludedIP.To4() != nil {
			return nil, errors.Errorf("specified prefix %s is not an IPv6 prefix", excludedPrefix)
		}

		pool.ExcludedPrefix = excludedNet.String()
	}

	return pool, nil
}

// Adds address pool to the database.
func AddAddressPool(db *dbops.PgDB, pool *AddressPool) error {
	if pool.LocalSubnetID == 0 && pool.LocalSubnet == nil {
		err := errors.Errorf("subnet must be specified when adding new pool %s-%s into the database",
			pool.LowerBound, pool.UpperBound)
		return err
	}

	// In case, the caller specified pointer to the subnet rather than subnet id
	// we have to set the subnet id on our own.
	if pool.LocalSubnetID == 0 && pool.LocalSubnet != nil {
		pool.LocalSubnetID = pool.LocalSubnet.ID
	}

	_, err := db.Model(pool).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem adding new address pool %s-%s into the database for subnet %d",
			pool.LowerBound, pool.UpperBound, pool.LocalSubnetID)
	}
	return err
}

// Adds prefix pool to the database.
func AddPrefixPool(db *dbops.PgDB, pool *PrefixPool) error {
	if pool.LocalSubnetID == 0 && pool.LocalSubnet == nil {
		err := errors.Errorf("local subnet must be specified when adding new prefix pool %s into the database",
			pool.Prefix)
		return err
	}

	// In case, the caller specified pointer to the subnet rather than subnet id
	// we have to set the subnet id on our own.
	if pool.LocalSubnetID == 0 && pool.LocalSubnet != nil {
		pool.LocalSubnetID = pool.LocalSubnet.ID
	}

	_, err := db.Model(pool).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem adding new prefix pool %s into the database for local subnet %d",
			pool.Prefix, pool.LocalSubnetID)
	}
	return err
}

// Deletes IPv4 or IPv6 address pool from the database.
func DeleteAddressPool(db *dbops.PgDB, poolID int64) error {
	pool := &AddressPool{
		ID: poolID,
	}
	result, err := db.Model(pool).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem deleting the address pool with ID %d", poolID)
	} else if result.RowsAffected() <= 0 {
		err = errors.Wrapf(ErrNotExists, "pool with ID %d does not exist", poolID)
	}
	return err
}

// Deletes IPv6 address pool from the database.
func DeletePrefixPool(db *dbops.PgDB, poolID int64) error {
	pool := &PrefixPool{
		ID: poolID,
	}
	result, err := db.Model(pool).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem deleting the prefix pool with ID %d", poolID)
	} else if result.RowsAffected() <= 0 {
		err = errors.Wrapf(ErrNotExists, "pool with ID %d does not exist", poolID)
	}
	return err
}
