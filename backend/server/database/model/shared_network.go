package dbmodel

import (
	"github.com/go-pg/pg/v9"
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
}

// Adds new shared network to the database.
func AddSharedNetwork(db *dbops.PgDB, network *SharedNetwork) error {
	err := db.Insert(network)
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new shared network %s", network.Name)
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
