package dbmodel

import (
	errors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"

	"time"
)

// Reflects IPv4 or IPv6 address pool.
type AddressPool struct {
	ID         int64
	Created    time.Time
	LowerBound string
	UpperBound string
	SubnetID   int64
	Subnet     *Subnet
}

// Reflects IPv6 address pool.
type PrefixPool struct {
	ID           int64
	Created      time.Time
	Prefix       string
	DelegatedLen int
	SubnetID     int64
	Subnet       *Subnet
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
