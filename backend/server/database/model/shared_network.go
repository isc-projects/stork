package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	errors "github.com/pkg/errors"

	dbops "isc.org/stork/server/database"

	"time"
)

// A structure reflecting shared_network SQL table. This table holds
// information about DHCP shared networks. A shared netwok groups
// multiple subnets together.
type SharedNetwork struct {
	ID      int64
	Created time.Time
	Name    string

	Subnets []Subnet
}

// Adds new shared network to the database.
func AddSharedNetwork(dbIface interface{}, network *SharedNetwork) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = errors.WithMessagef(err, "problem with starting transaction for adding new shared network with name %s",
			network.Name)
		return err
	}
	defer rollback()

	err = tx.Insert(network)
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new shared network %s into the database", network.Name)
		return err
	}

	for i, s := range network.Subnets {
		subnet := s
		subnet.SharedNetworkID = network.ID

		err = addSubnetWithPools(tx, &subnet)
		if err != nil {
			return err
		}
		network.Subnets[i] = subnet
	}

	err = commit()
	if err != nil {
		err = errors.WithMessagef(err, "problem with committing new shared network with name %s into the database",
			network.Name)
	}

	return err
}

// Updates shared network in the database. It neither adds nor modifies associations
// with the subnets it contains.
func UpdateSharedNetwork(dbIface interface{}, network *SharedNetwork) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = errors.WithMessagef(err, "problem with starting transaction for updating shared network with name %s",
			network.Name)
		return err
	}
	defer rollback()

	err = tx.Update(network)
	if err != nil {
		err = errors.Wrapf(err, "problem with updating the shared network with id %d", network.ID)
		return err
	}

	err = commit()
	if err != nil {
		err = errors.WithMessagef(err, "problem with committing updates to shared network with name %s into the database",
			network.Name)
	}
	return err
}

// Fetches all shared networks without subnets.
func GetAllSharedNetworks(db *dbops.PgDB) ([]SharedNetwork, error) {
	networks := []SharedNetwork{}
	err := db.Model(&networks).
		OrderExpr("id ASC").
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return []SharedNetwork{}, nil
		}
		err = errors.Wrapf(err, "problem with getting all shared networks")
		return nil, err
	}
	return networks, err
}

// Fetches the information about the selected shared network.
func GetSharedNetwork(db *dbops.PgDB, networkID int64) (*SharedNetwork, error) {
	network := &SharedNetwork{}
	err := db.Model(network).
		Where("shared_network.id = ?", networkID).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a shared network with id %d", networkID)
		return nil, err
	}
	return network, err
}

// Fetches a shared network with the subnets it contains.
func GetSharedNetworkWithSubnets(db *dbops.PgDB, networkID int64) (network *SharedNetwork, err error) {
	network = &SharedNetwork{}
	err = db.Model(network).
		Relation("Subnets").
		Relation("Subnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("Subnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("Subnets.LocalSubnets.App").
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a shared network with id %d and its subnets", networkID)
		return nil, err
	}
	return network, err
}

// Deletes the selected shared network from the database.
func DeleteSharedNetwork(db *dbops.PgDB, networkID int64) error {
	network := &SharedNetwork{
		ID: networkID,
	}
	_, err := db.Model(network).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting the shared network with id %d", networkID)
	}
	return err
}

// Deletes a shared network and along with its subnets.
func DeleteSharedNetworkWithSubnets(db *dbops.PgDB, networkID int64) error {
	tx, err := db.Begin()
	if err != nil {
		err = errors.Wrapf(err, "problem with starting transaction for deleting shared network with id %d and its subnets",
			networkID)
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete all subnets blonging to the shared network.
	subnets := []Subnet{}
	_, err = db.Model(&subnets).
		Where("subnet.shared_network_id = ?", networkID).
		Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting subnets from the shared network with id %d", networkID)
		return err
	}

	// If everything went fine, delete the shared network. Note that shared network
	// does not trigger cascaded deletion of the subnets.
	network := &SharedNetwork{
		ID: networkID,
	}
	_, err = db.Model(network).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting the shared network with id %d", networkID)
		return err
	}

	err = tx.Commit()
	if err != nil {
		err = errors.Wrapf(err, "problem with committing deleted shared network with id %d",
			networkID)
	}
	return err
}
