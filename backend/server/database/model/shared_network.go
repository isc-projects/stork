package dbmodel

import (
	"context"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// A structure reflecting shared_network SQL table. This table holds
// information about DHCP shared networks. A shared network groups
// multiple subnets together.
type SharedNetwork struct {
	ID        int64
	CreatedAt time.Time
	Name      string
	Family    int `pg:"inet_family"`

	Subnets []Subnet `pg:"rel:has-many"`

	AddrUtilization  int16
	PdUtilization    int16
	Stats            SubnetStats
	StatsCollectedAt time.Time
}

// Adds new shared network to the database in a transaction.
func addSharedNetwork(tx *pg.Tx, network *SharedNetwork) error {
	_, err := tx.Model(network).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new shared network %s to the database", network.Name)
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
	return nil
}

// Adds new shared network to the database. It begins a new transaction
// when dbi has a *pg.DB type or uses an existing transaction when dbi
// has a *pg.Tx type.
func AddSharedNetwork(dbi dbops.DBI, network *SharedNetwork) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addSharedNetwork(tx, network)
		})
	}
	return addSharedNetwork(dbi.(*pg.Tx), network)
}

// Updates shared network in the database in a transaction. It neither adds
// nor modifies associations with the subnets it contains.
func updateSharedNetwork(tx *pg.Tx, network *SharedNetwork) error {
	result, err := tx.Model(network).WherePK().Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating the shared network with ID %d", network.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "shared network with ID %d does not exist", network.ID)
	}
	return err
}

// Updates shared network in the database. It neither adds nor modifies
// associations with the subnets it contains. It begins a new transaction
// when dbi has a *pg.DB type or uses an existing transaction when dbi
// has a *pg.Tx type.
func UpdateSharedNetwork(dbi dbops.DBI, network *SharedNetwork) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateSharedNetwork(tx, network)
		})
	}
	return updateSharedNetwork(dbi.(*pg.Tx), network)
}

// Fetches all shared networks without subnets. The family argument specifies
// whether only IPv4 shared networks should be fetched (if 4), only IPv6 shared
// networks should be fetched (if 6) or both otherwise.
func GetAllSharedNetworks(dbi dbops.DBI, family int) ([]SharedNetwork, error) {
	networks := []SharedNetwork{}
	q := dbi.Model(&networks)

	if family == 4 || family == 6 {
		q = q.Where("inet_family = ?", family)
	}
	q = q.OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []SharedNetwork{}, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all shared networks")
		return nil, err
	}
	return networks, err
}

// Fetches the information about the selected shared network.
func GetSharedNetwork(dbi dbops.DBI, networkID int64) (*SharedNetwork, error) {
	network := &SharedNetwork{}
	err := dbi.Model(network).
		Where("shared_network.id = ?", networkID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting the shared network with ID %d", networkID)
		return nil, err
	}
	return network, err
}

// Fetches a shared network with the subnets it contains.
func GetSharedNetworkWithSubnets(dbi dbops.DBI, networkID int64) (network *SharedNetwork, err error) {
	network = &SharedNetwork{}
	err = dbi.Model(network).
		Relation("Subnets").
		Relation("Subnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("Subnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("Subnets.LocalSubnets.Daemon.App.AccessPoints").
		Where("shared_network.id = ?", networkID).
		Select()

	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting the shared network with ID %d and its subnets", networkID)
		return nil, err
	}
	return network, err
}

// Deletes the selected shared network from the database.
func DeleteSharedNetwork(dbi dbops.DBI, networkID int64) error {
	network := &SharedNetwork{
		ID: networkID,
	}
	result, err := dbi.Model(network).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the shared network with ID %d", networkID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "shared network with ID %d does not exist", networkID)
	}
	return err
}

// Deletes a shared network and along with its subnets.
func DeleteSharedNetworkWithSubnets(db *dbops.PgDB, networkID int64) (err error) {
	tx, err := db.Begin()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem starting transaction to delete shared network with ID %d and its subnets",
			networkID)
		return
	}
	defer dbops.RollbackOnError(tx, &err)

	// Delete all subnets belonging to the shared network.
	subnets := []Subnet{}
	_, err = db.Model(&subnets).
		Where("subnet.shared_network_id = ?", networkID).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting subnets from the shared network with ID %d", networkID)
		return err
	}

	// If everything went fine, delete the shared network. Note that shared network
	// does not trigger cascaded deletion of the subnets.
	network := &SharedNetwork{
		ID: networkID,
	}
	result, err := db.Model(network).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the shared network with ID %d", networkID)
		return
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "shared network with ID %d does not exist", networkID)
		return
	}

	err = tx.Commit()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem committing deleted shared network with ID %d",
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
func GetSharedNetworksByPage(dbi dbops.DBI, offset, limit, appID, family int64, filterText *string, sortField string, sortDir SortDirEnum) ([]SharedNetwork, int64, error) {
	networks := []SharedNetwork{}
	q := dbi.Model(&networks)

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
	// When filtering by appID we also need the daemon table (via joined local_subnet)
	// as it holds the app identifier.
	if appID != 0 {
		q = q.Join("INNER JOIN local_subnet AS ls ON s.id = ls.subnet_id")
		q = q.Join("INNER JOIN daemon AS d ON d.id = ls.daemon_id")
	}
	// Include address pools, prefix pools and the local subnet info in the results.
	q = q.Relation("Subnets.AddressPools", func(q *orm.Query) (*orm.Query, error) {
		return q.Order("address_pool.id ASC"), nil
	}).
		Relation("Subnets.PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("Subnets.LocalSubnets.Daemon.App.AccessPoints").
		Relation("Subnets.LocalSubnets.Daemon.App.Machine")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(s.prefix) = ?", family)
	}

	// Filter by appID.
	if appID != 0 {
		q = q.Where("d.app_id = ?", appID)
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
		err = pkgerrors.Wrapf(err, "problem getting shared networks by page")
	}
	return networks, int64(total), err
}

// Update statistics and utilization of addresses and delegated prefixes in a SharedNetwork.
func UpdateStatisticsInSharedNetwork(dbi dbops.DBI, sharedNetworkID int64, statistics SubnetStats) error {
	addrUtilization, pdUtilization := statistics.GetUtilizations()
	net := &SharedNetwork{
		ID:               sharedNetworkID,
		AddrUtilization:  int16(addrUtilization * 1000),
		PdUtilization:    int16(pdUtilization * 1000),
		Stats:            statistics,
		StatsCollectedAt: time.Now().UTC(),
	}
	q := dbi.Model(net)
	q = q.Column("addr_utilization", "pd_utilization", "stats", "stats_collected_at")
	q = q.WherePK()
	result, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating utilization in the shared network: %d",
			sharedNetworkID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "shared network with ID %d does not exist", sharedNetworkID)
	}
	return err
}

// Deletes shared networks which include no subnets. Returns deleted shared networks
// count and an error.
func DeleteEmptySharedNetworks(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]Subnet{}).
		Column("id").
		Limit(1).
		Where("shared_network.id = subnet.shared_network_id")
	result, err := dbi.Model(&[]SharedNetwork{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting empty shared networks")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}
