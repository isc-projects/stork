package dbmodel

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// This structure holds subnet information retrieved from an app. Multiple
// DHCP server apps may be configured to serve leases in the same subnet.
// For the same subnet configured on different DHCP server there will be
// a separate instance of the LocalSubnet structure. Apart from possibly
// different local subnet id between different apos there will also be
// other information stored here, e.g. statistics for the particular
// subnet retrieved from the given app. Multiple local subnets can be
// associated with a single global subnet depending on how many apps serve
// the same subnet.
type LocalSubnet struct {
	AppID         int64 `pg:",pk"`
	SubnetID      int64 `pg:",pk"`
	App           *App
	Subnet        *Subnet
	LocalSubnetID int64

	Stats            map[string]interface{}
	StatsCollectedAt time.Time
}

// Reflects IPv4 or IPv6 subnet from the database.
type Subnet struct {
	ID          int64
	CreatedAt   time.Time
	Prefix      string
	ClientClass string

	SharedNetworkID int64
	SharedNetwork   *SharedNetwork

	AddressPools []AddressPool
	PrefixPools  []PrefixPool

	LocalSubnets []*LocalSubnet

	Hosts []Host

	AddrUtilization int16
	PdUtilization   int16
}

// Hook executed after inserting a subnet to the database. It updates subnet
// id on the hosts belonging to this subnet.
func (s *Subnet) AfterInsert(ctx context.Context) error {
	if s != nil && s.ID != 0 {
		for i := range s.Hosts {
			s.Hosts[i].SubnetID = s.ID
		}
	}
	return nil
}

// Return family of the subnet.
func (s *Subnet) GetFamily() int {
	family := 4
	if strings.Contains(s.Prefix, ":") {
		family = 6
	}
	return family
}

// Add address and prefix pools from the subnet instance into the database.
// The subnet is expected to exist in the database. The dbIface argument
// is either a pointer to pg.DB if the new transaction should be started
// by this function or it is a pointer to pg.Tx which represents an
// existing transaction. That way this function can be called to add
// pools within a separate transaction or as part of an existing
// transaction.
func addSubnetPools(dbIface interface{}, subnet *Subnet) (err error) {
	if len(subnet.AddressPools) == 0 && len(subnet.PrefixPools) == 0 {
		return nil
	}

	// This function is meant to be used both within a transaction and to
	// create its own transaction. Depending on the object type, we either
	// use the existing transaction or start the new one.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding pools to subnet with id %d",
			subnet.ID)
	}
	defer rollback()

	// Add address pools first.
	for i, p := range subnet.AddressPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem with adding an address pool %s-%s for subnet with id %d",
				pool.LowerBound, pool.UpperBound, subnet.ID)
			return err
		}
		subnet.AddressPools[i] = pool
	}
	// Add prefix pools. This should be empty for IPv4 case.
	for i, p := range subnet.PrefixPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem with adding a prefix pool %s for subnet with id %d",
				pool.Prefix, subnet.ID)
			return err
		}
		subnet.PrefixPools[i] = pool
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing pools into a subnet with id %d", subnet.ID)
	}

	return err
}

// Adds a new subnet and its pools to the database within a transaction.
func addSubnetWithPools(tx *pg.Tx, subnet *Subnet) error {
	// Add the subnet first.
	_, err := tx.Model(subnet).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding new subnet with prefix %s", subnet.Prefix)
		return err
	}

	// Add the pools.
	err = addSubnetPools(tx, subnet)
	if err != nil {
		return err
	}
	return err
}

// Creates new transaction and adds the subnet along with its pools into the
// database. If it has any associations with the shared network, those
// associations are also made in the database.
func AddSubnet(dbIface interface{}, subnet *Subnet) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding new subnet with prefix %s",
			subnet.Prefix)
		return err
	}
	defer rollback()

	err = addSubnetWithPools(tx, subnet)
	if err != nil {
		return err
	}
	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing new subnet with prefix %s into the database",
			subnet.Prefix)
	}

	return err
}

// Fetches the subnet and its pools by id from the database.
func GetSubnet(db *pg.DB, subnetID int64) (*Subnet, error) {
	subnet := &Subnet{}
	err := db.Model(subnet).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints").
		Where("subnet.id = ?", subnetID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting a subnet with id %d", subnetID)
		return nil, err
	}
	return subnet, err
}

// Fetches the subnet and its pools by local subnet ID and application ID.
// Applications may use their own numbering scheme for subnets. Therefore,
// the local subnet ID must be accompanied by the app ID.
// If the family is set to 0 it fetches both IPv4 and IPv6 subnets.
// Use 4 o 6 to fetch IPv4 or IPv6 specific subnet.
func GetSubnetsByLocalID(db *pg.DB, localSubnetID int64, appID int64, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := db.Model(&subnets).
		Join("INNER JOIN local_subnet AS ls ON ls.local_subnet_id = ? AND ls.app_id = ?", localSubnetID, appID).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints")

	// Optionally filter by IPv4 or IPv6 subnets.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting subnets by local subnet id %d and app id %d", localSubnetID, appID)
		return nil, err
	}
	return subnets, err
}

// Fetches all subnets associated with the given application by ID.
// If the family is set to 0 it fetches both IPv4 and IPv6 subnets.
// Use 4 o 6 to fetch IPv4 or IPv6 specific subnet.
func GetSubnetsByAppID(db *pg.DB, appID int64, family int) ([]Subnet, error) {
	subnets := []Subnet{}

	q := db.Model(&subnets).
		Join("INNER JOIN local_subnet AS ls ON ls.subnet_id = subnet.id").
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints").
		Where("ls.app_id = ?", appID)

	// Optionally filter by IPv4 or IPv6 subnets.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting subnets by app id %d", appID)
		return nil, err
	}
	return subnets, err
}

// Fetches the subnet by prefix from the database.
func GetSubnetsByPrefix(db *pg.DB, prefix string) ([]Subnet, error) {
	subnets := []Subnet{}
	err := db.Model(&subnets).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints").
		Where("subnet.prefix = ?", prefix).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting subnets with prefix %s", prefix)
		return nil, err
	}
	return subnets, err
}

// Fetches all subnets belonging to a given family. If the family is set to 0
// it fetches both IPv4 and IPv6 subnet.
func GetAllSubnets(db *pg.DB, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := db.Model(&subnets).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints").
		OrderExpr("id ASC")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting all subnets for family %d", family)
		return nil, err
	}
	return subnets, err
}

// Fetches a collection of subnets from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The appID is used to filter subnets to those handled by the
// given application.  The family is used to filter by IPv4 (if 4) or
// IPv6 (if 6). For all other values of the family parameter both IPv4
// and IPv6 subnets are returned. The filterText can be used to match
// the subnet prefix or pool ranges. The nil value disables such
// filtering. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting.  in SortDirAny is used then ASC
// order is used. This function returns a collection of subnets, the
// total number of subnets and error.
func GetSubnetsByPage(db *pg.DB, offset, limit, appID, family int64, filterText *string, sortField string, sortDir SortDirEnum) ([]Subnet, int64, error) {
	subnets := []Subnet{}
	q := db.Model(&subnets).Distinct()

	// When filtering by appID we also need the local_subnet table as it holds the
	// application identifier.
	if appID != 0 {
		q = q.Join("INNER JOIN local_subnet AS ls ON subnet.id = ls.subnet_id")
	}
	// Pools are also required when trying to filter by text.
	if filterText != nil {
		q = q.Join("LEFT JOIN address_pool AS ap ON subnet.id = ap.subnet_id")
	}
	// Include pools, shared network the subnets belong to, local subnet info
	// and the associated apps in the results.
	q = q.Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
		return q.Order("address_pool.id ASC"), nil
	}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.App.AccessPoints").
		Relation("LocalSubnets.App.Machine")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}

	// Filter by appID.
	if appID != 0 {
		q = q.Where("ls.app_id = ?", appID)
	}

	// Quick filtering by subnet prefix, pool ranges or shared network name.
	if filterText != nil {
		// The combination of the concat and host functions reconstruct the textual
		// version of the pool range as specified in Kea, e.g. 192.0.2.10-192.0.2.20.
		// This allows for quick filtering by strings like: 2.10-192.0.
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(subnet.prefix) LIKE ?", "%"+*filterText+"%").
				WhereOr("concat(host(ap.lower_bound), '-', host(ap.upper_bound)) LIKE ?", "%"+*filterText+"%").
				WhereOr("shared_network.name LIKE ?", "%"+*filterText+"%")
			return q, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("subnet", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	// This returns the limited results plus the total number of records.
	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting subnets by page")
	}
	return subnets, int64(total), err
}

// Get list of Subnets with LocalSubnets ordered by SharedNetworkID.
func GetSubnetsWithLocalSubnets(db *pg.DB) ([]*Subnet, error) {
	subnets := []*Subnet{}
	q := db.Model(&subnets)
	// only selected columns are returned for performance reasons
	q = q.Column("id", "shared_network_id", "prefix")
	q = q.Relation("LocalSubnets")
	q = q.Order("shared_network_id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrap(err, "problem with getting all subnets")
		return nil, err
	}
	return subnets, nil
}

// Associates an application with the subnet having a specified ID and prefix.
// Internally, the association is made via the local_subnet table which holds
// information about the subnet from the given app perspective, local subnet
// id, statistics etc.
func AddAppToSubnet(dbIface interface{}, subnet *Subnet, app *App) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for associating an app with id %d with the subnet %s",
			app.ID, subnet.Prefix)
		return err
	}
	defer rollback()

	localSubnetID := int64(0)
	// If the prefix is available we should try to match the subnet prefix
	// with the app's configuration and retrieve the local subnet id from
	// there.
	if len(subnet.Prefix) > 0 {
		localSubnetID = app.GetLocalSubnetID(subnet.Prefix)
	}
	localSubnet := LocalSubnet{
		AppID:         app.ID,
		SubnetID:      subnet.ID,
		LocalSubnetID: localSubnetID,
	}
	// Try to insert. If such association already exists we could maybe do
	// nothing, but we do update instead to force setting the new value
	// of the local_subnet_id if it has changed.
	_, err = tx.Model(&localSubnet).
		Column("app_id").
		Column("subnet_id").
		Column("local_subnet_id").
		OnConflict("(app_id, subnet_id) DO UPDATE").
		Set("local_subnet_id = EXCLUDED.local_subnet_id").
		Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with associating the app with id %d with the subnet %s",
			app.ID, subnet.Prefix)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing transaction associating the app with id %d with the subnet %s",
			app.ID, subnet.Prefix)
	}
	return err
}

// Dissociates an application from the subnet having a specified id.
// The first returned value indicates if any row was removed from the
// daemon_to_service table.
func DeleteAppFromSubnet(db *pg.DB, subnetID int64, appID int64) (bool, error) {
	localSubnet := &LocalSubnet{
		AppID:    appID,
		SubnetID: subnetID,
	}
	rows, err := db.Model(localSubnet).WherePK().Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem with deleting an app with id %d from the subnet with %d",
			appID, subnetID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Finds and returns an app associated with a subnet having the specified id.
func (s *Subnet) GetApp(appID int64) *App {
	for _, s := range s.LocalSubnets {
		app := s.App
		if app.ID == appID {
			return app
		}
	}
	return nil
}

// Iterates over the provided slice of subnets and stores them in the database
// if they are not there yet. In addition, it associates the subnets with the
// specified Kea application. Returns a list of added subnets.
func commitSubnetsIntoDB(tx *pg.Tx, networkID int64, subnets []Subnet, app *App, seq int64) (addedSubnets []*Subnet, err error) {
	for i := range subnets {
		subnet := &subnets[i]
		if subnet.ID == 0 {
			subnet.SharedNetworkID = networkID
			err = AddSubnet(tx, subnet)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected subnet %s to the database",
					subnet.Prefix)
				return nil, err
			}
			addedSubnets = append(addedSubnets, subnet)
		}
		err = AddAppToSubnet(tx, subnet, app)
		if err != nil {
			err = pkgerrors.WithMessagef(err, "unable to associate detected subnet %s with Kea app having id %d", subnet.Prefix, app.ID)
			return nil, err
		}

		err = CommitSubnetHostsIntoDB(tx, subnet, app, "config", seq)
		if err != nil {
			return nil, err
		}
	}
	return addedSubnets, nil
}

// Iterates over the shared networks, subnets and hosts and commits them to the database.
// In addition it associates them with the specified app. Returns a list of added subnets.
func CommitNetworksIntoDB(dbIface interface{}, networks []SharedNetwork, subnets []Subnet, app *App, seq int64) ([]*Subnet, error) {
	// Begin transaction.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return nil, err
	}
	defer rollback()

	var addedSubnets []*Subnet
	var addedSubnetsToNet []*Subnet

	// Go over the networks that the Kea app belongs to.
	for i := range networks {
		network := &networks[i]
		if network.ID == 0 {
			// This is new shared network. Add it to the database.
			err = AddSharedNetwork(tx, network)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected shared network %s to the database",
					network.Name)
				return nil, err
			}
		}
		// Associate subnets with the app.
		addedSubnetsToNet, err = commitSubnetsIntoDB(tx, network.ID, network.Subnets, app, seq)
		if err != nil {
			return nil, err
		}
		addedSubnets = append(addedSubnets, addedSubnetsToNet...)
	}

	// Finally, add top level subnets to the database and associate them with
	// the Kea app.
	addedSubnetsToNet, err = commitSubnetsIntoDB(tx, 0, subnets, app, seq)
	if err != nil {
		return nil, err
	}
	addedSubnets = append(addedSubnets, addedSubnetsToNet...)

	// Commit the changes if everything went fine.
	err = commit()
	return addedSubnets, err
}

// Fetch all local subnets for indicated app.
func GetAppLocalSubnets(db *pg.DB, appID int64) ([]*LocalSubnet, error) {
	subnets := []*LocalSubnet{}
	q := db.Model(&subnets)
	// only selected columns are returned while stats columns are skipped for performance reasons (they are pretty big json fields)
	q = q.Column("app_id", "subnet_id", "local_subnet_id")
	q = q.Relation("Subnet")
	q = q.Where("local_subnet.app_id = ?", appID)

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting all local subnets for app %d", appID)
		return nil, err
	}
	return subnets, nil
}

// Update stats pulled for given local subnet.
func (lsn *LocalSubnet) UpdateStats(db *pg.DB, stats map[string]interface{}) error {
	lsn.Stats = stats
	lsn.StatsCollectedAt = storkutil.UTCNow()
	q := db.Model(lsn)
	q = q.Column("stats", "stats_collected_at")
	q = q.WherePK()
	_, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating stats in local subnet: [app:%d, subnet:%d, local subnet:%d]",
			lsn.AppID, lsn.SubnetID, lsn.LocalSubnetID)
	}
	return err
}

// Update utilization in Subnet.
func (s *Subnet) UpdateUtilization(db *pg.DB, addrUtilization, pdUtilization int16) error {
	s.AddrUtilization = addrUtilization
	s.PdUtilization = pdUtilization
	q := db.Model(s)
	q = q.Column("addr_utilization", "pd_utilization")
	q = q.WherePK()
	_, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating utilization in the subnet: %d",
			s.ID)
	}
	return err
}
