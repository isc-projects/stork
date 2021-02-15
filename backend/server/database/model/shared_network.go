package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// A structure reflecting shared_network SQL table. This table holds
// information about DHCP shared networks. A shared netwok groups
// multiple subnets together.
type SharedNetwork struct {
	ID        int64
	CreatedAt time.Time
	Name      string
	Family    int `pg:"inet_family"`

	Subnets []Subnet

	AddrUtilization int16
	PdUtilization   int16
}

// Adds new shared network to the database.
func AddSharedNetwork(dbIface interface{}, network *SharedNetwork) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding new shared network with name %s",
			network.Name)
		return err
	}
	defer rollback()

	err = tx.Insert(network)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding new shared network %s into the database", network.Name)
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
		err = pkgerrors.WithMessagef(err, "problem with committing new shared network with name %s into the database",
			network.Name)
	}

	return err
}

// Updates shared network in the database. It neither adds nor modifies associations
// with the subnets it contains.
func UpdateSharedNetwork(dbIface interface{}, network *SharedNetwork) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for updating shared network with name %s",
			network.Name)
		return err
	}
	defer rollback()

	err = tx.Update(network)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating the shared network with id %d", network.ID)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing updates to shared network with name %s into the database",
			network.Name)
	}
	return err
}

// Fetches all shared networks without subnets. The family argument specifies
// whether only IPv4 shared networks should be fetched (if 4), only IPv6 shared
// networks should be fetched (if 6) or both otherwise.
func GetAllSharedNetworks(db *dbops.PgDB, family int) ([]SharedNetwork, error) {
	networks := []SharedNetwork{}
	q := db.Model(&networks)

	if family == 4 || family == 6 {
		q = q.Where("inet_family = ?", family)
	}
	q = q.OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []SharedNetwork{}, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting all shared networks")
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
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting a shared network with id %d", networkID)
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
		Relation("Subnets.LocalSubnets.App.AccessPoints").
		Where("shared_network.id = ?", networkID).
		Select()

	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting a shared network with id %d and its subnets", networkID)
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
		err = pkgerrors.Wrapf(err, "problem with deleting the shared network with id %d", networkID)
	}
	return err
}

// Deletes a shared network and along with its subnets.
func DeleteSharedNetworkWithSubnets(db *dbops.PgDB, networkID int64) error {
	tx, err := db.Begin()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with starting transaction for deleting shared network with id %d and its subnets",
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
		err = pkgerrors.Wrapf(err, "problem with deleting subnets from the shared network with id %d", networkID)
		return err
	}

	// If everything went fine, delete the shared network. Note that shared network
	// does not trigger cascaded deletion of the subnets.
	network := &SharedNetwork{
		ID: networkID,
	}
	_, err = db.Model(network).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting the shared network with id %d", networkID)
		return err
	}

	err = tx.Commit()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with committing deleted shared network with id %d",
			networkID)
	}
	return err
}

// Fetches a collection of shared networks from the database. The
// offset and limit specify the beginning of the page and the maximum
// size of the page. The appID is used to filter shared networks to
// those handled by the given application.  The family is used to
// filter by IPv4 (if 4) or IPv6 (if 6). For all other values of the
// family parameter both IPv4 and IPv6 shared networks are
// returned. The filterText can be used to match the shared network
// name or subnet prefix. The nil value disables such
// filtering. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting.  in SortDirAny is used then ASC
// order is used. This function returns a collection of shared
// networks, the total number of shared networks and error.
func GetSharedNetworksByPage(db *pg.DB, offset, limit, appID, family int64, filterText *string, sortField string, sortDir SortDirEnum) ([]SharedNetwork, int64, error) {
	networks := []SharedNetwork{}
	q := db.Model(&networks)

	// prepare distinct on expression to include sort field, otherwise distinct on will fail
	distingOnFields := "shared_network.id"
	if sortField != "" && sortField != "id" && sortField != "shared_network.id" {
		distingOnFields = sortField + ", " + distingOnFields
	}
	q = q.DistinctOn(distingOnFields)

	// If any of the filtering parameters are specified we need to explicitly join
	// the subnets table so as we can access its columns in the Where clause.
	if appID != 0 || family != 0 || filterText != nil {
		q = q.Join("INNER JOIN subnet AS s ON shared_network.id = s.shared_network_id")
	}
	// When filtering by appID we also need the local_subnet table as it holds the
	// application identifier.
	if appID != 0 {
		q = q.Join("INNER JOIN local_subnet AS ls ON s.id = ls.subnet_id")
	}
	// Include address pools, prefix pools and the local subnet info in the results.
	q = q.Relation("Subnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
		return q.Order("address_pool.id ASC"), nil
	}).
		Relation("Subnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("Subnets.LocalSubnets.App.AccessPoints").
		Relation("Subnets.LocalSubnets.App.Machine")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(s.prefix) = ?", family)
	}

	// Filter by appID.
	if appID != 0 {
		q = q.Where("ls.app_id = ?", appID)
	}

	// Quick filtering by shared network name or subnet prefix.
	if filterText != nil {
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("TEXT(s.prefix) LIKE ?", "%"+*filterText+"%").
				WhereOr("shared_network.name LIKE ?", "%"+*filterText+"%")
			return q, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("shared_network", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	// This returns the limited results plus the total number of records.
	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting shared networks by page")
	}
	return networks, int64(total), err
}

// Update utilization of addresses and delegated prefixes in a SharedNetwork.
func UpdateUtilizationInSharedNetwork(db *pg.DB, sharedNetworkID int64, addrUtilization, pdUtilization int16) error {
	net := &SharedNetwork{
		ID:              sharedNetworkID,
		AddrUtilization: addrUtilization,
		PdUtilization:   pdUtilization,
	}
	q := db.Model(net)
	q = q.Column("addr_utilization", "pd_utilization")
	q = q.WherePK()
	_, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating utilization in the shared network: %d",
			sharedNetworkID)
	}
	return err
}
