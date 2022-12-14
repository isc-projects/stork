package dbmodel

import (
	"net"
	"time"

	errors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Reflects IPv4 or IPv6 address pool.
type AddressPool struct {
	ID         int64
	CreatedAt  time.Time
	LowerBound string
	UpperBound string
	SubnetID   int64
	Subnet     *Subnet `pg:"rel:has-one"`
}

// Reflects IPv6 prefix pool.
type PrefixPool struct {
	ID             int64
	CreatedAt      time.Time
	Prefix         string
	DelegatedLen   int
	ExcludedPrefix string
	SubnetID       int64
	Subnet         *Subnet `pg:"rel:has-one"`
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

	pool := &PrefixPool{}
	pool.Prefix = prefixNet.String()
	pool.DelegatedLen = delegatedLen

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
	if pool.SubnetID == 0 && pool.Subnet == nil {
		err := errors.Errorf("subnet must be specified when adding new pool %s-%s into the database",
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
		err = errors.Wrapf(err, "problem adding new address pool %s-%s into the database for subnet %d",
			pool.LowerBound, pool.UpperBound, pool.SubnetID)
	}
	return err
}

// Adds prefix pool to the database.
func AddPrefixPool(db *dbops.PgDB, pool *PrefixPool) error {
	if pool.Subnet.ID == 0 && pool.Subnet == nil {
		err := errors.Errorf("subnet must be specified when adding new prefix pool %s into the database",
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
		err = errors.Wrapf(err, "problem adding new prefix pool %s into the database for subnet %d",
			pool.Prefix, pool.SubnetID)
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
