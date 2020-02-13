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
func AddSharedNetwork(db *dbops.PgDB, network *SharedNetwork) error {
	tx, err := db.Begin()
	if err != nil {
		err = errors.Wrapf(err, "problem with starting transaction for adding new shared network with name %s",
			network.Name)
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = tx.Insert(network)
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new shared network %s into the database", network.Name)
		return err
	}

	for _, s := range network.Subnets {
		subnet := s
		subnet.SharedNetworkID = network.ID

		err = addSubnetWithPools(tx, &subnet)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		err = errors.Wrapf(err, "problem with coommitting new shared network with name %s into the database",
			network.Name)
	}

	return err
}

// Updates shared network in the database. It neither adds nor modifies associations
// with the subnets it contains.
func UpdateSharedNetwork(db *dbops.PgDB, network *SharedNetwork) error {
	err := db.Update(network)
	if err != nil {
		err = errors.Wrapf(err, "problem with updating the shared network with id %d", network.ID)
	}
	return err
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
	subnets := []Subnet{}

	// The query we're building makes a select against subnets rather than shared networks
	// because it is super complicated (if possible) to use ORM to make a "cascade" query
	// to fetch 3 levels of information shared networks->subnets->pools. If you query for
	// subnets you can easily join both shared networks and pools.
	err = db.Model(&subnets).
		Relation("SharedNetwork", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("shared_network.id = ?", networkID).
				OrderExpr("shared_network.id ASC"), nil
		}).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.OrderExpr("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.OrderExpr("prefix_pool.id ASC"), nil
		}).
		Select()

	// If there was nothing returned, it doesn't mean that there is no shared network.
	// It merely means there are no subnets belonging to it (which is rare).
	// If that's the case, simply get the shared network.
	if err == pg.ErrNoRows {
		network, err = GetSharedNetwork(db, networkID)
	} else {
		// Subnets with the shared network have been returned. Let's create the
		// shared network instance and attach the returned subnets to it. Take
		// the subnet instance from the first subnet we found. We could take
		// it from any subnet actually.
		network = subnets[0].SharedNetwork
		network.Subnets = subnets
	}
	if err != nil {
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
