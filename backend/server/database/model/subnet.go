package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	errors "github.com/pkg/errors"

	"time"
)

// This structure reflects an app which is associated with a subnet in M:N
// relationship. Such association is made via the app_to_subnet table which,
// besides the primary key, includes the optional local subnet id value.
// This struct embeds the app and glues it with the local subnet id. The
// value of the local subnet id is copied via App_LocalSubnetID field of the
// AppToSubnet structure. Without this structure, the value of the local
// subnet id would not be available within the Subnet structure after
// querying for a subnet with associated apps. The LocalSubnetID value
// is copied to this structure thanks to using App_LocalSubnetID in the
// AppToSubnet structure which, when present, signals to go-pg to capture
// this value.
type SubnetAttachedApp struct {
	tableName struct{} `pg:"app"` //nolint:unused,structcheck
	App
	LocalSubnetID int64 `pg:"-"`
}

// Reflects IPv4 or IPv6 subnet from the database.
type Subnet struct {
	ID      int64
	Created time.Time
	Prefix  string

	SharedNetworkID int64
	SharedNetwork   *SharedNetwork

	AddressPools []AddressPool
	PrefixPools  []PrefixPool

	Apps []*SubnetAttachedApp `pg:"many2many:app_to_subnet,fk:subnet_id,joinFK:app_id"`
}

// A structure reflecting an app_to_subnet SQL table which associates
// applications with subnets in many to many relationship. It also
// provides additional association of the global subnet id (from the
// database) and optional application specific subnet id (local one).
// The Kea specific subnet id is an example of the local subnet id.
// The local subnet id is non-unique, because different app instances
// may use the same id, even for different subnets. In fact, the same
// app may use the same local subnet ID twice, once for DHCPv4 and
// second time for DHCPv6.
// Note that App_LocalSubnetID is used in queries to copy the value of
// the local subnet id to the SubnetAttachedApp structure. The
// LocalSubnetID is used in statements inserting the data to the
// app_to_subnet table.
type AppToSubnet struct {
	AppID             int64 `pg:",pk"`
	SubnetID          int64 `pg:",pk"`
	App_LocalSubnetID int64 //nolint:golint,stylecheck
	LocalSubnetID     int64
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
	var tx *pg.Tx
	db, ok := dbIface.(*pg.DB)
	if ok {
		tx, err = db.Begin()
		if err != nil {
			err = errors.Wrapf(err, "problem with starting transaction for adding pools to subnet with id %d",
				subnet.ID)
		}
		defer func() {
			_ = tx.Rollback()
		}()
	} else {
		tx, ok = dbIface.(*pg.Tx)
		if !ok {
			err = errors.New("unsupported type of the database transaction object provided")
			return err
		}
	}

	// Add address pools first.
	for i, p := range subnet.AddressPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
		if err != nil {
			err = errors.Wrapf(err, "problem with adding an address pool %s-%s for subnet with id %d",
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
			err = errors.Wrapf(err, "problem with adding a prefix pool %s for subnet with id %d",
				pool.Prefix, subnet.ID)
			return err
		}
		subnet.PrefixPools[i] = pool
	}

	if db != nil {
		err = tx.Commit()
		if err != nil {
			err = errors.Wrapf(err, "problem with committing pools into a subnet with id %d", subnet.ID)
		}
	}

	return err
}

// Adds a new subnet and its pools to the database within a transaction.
func addSubnetWithPools(tx *pg.Tx, subnet *Subnet) error {
	// Add the subnet first.
	_, err := tx.Model(subnet).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new subnet with prefix %s", subnet.Prefix)
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
func AddSubnet(db *pg.DB, subnet *Subnet) error {
	tx, err := db.Begin()
	if err != nil {
		err = errors.Wrapf(err, "problem with starting transaction for adding new subnet with prefix %s",
			subnet.Prefix)
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = addSubnetWithPools(tx, subnet)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		err = errors.Wrapf(err, "problem with committing new subnet with prefix %s into the database",
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
		Relation("Apps").
		Where("subnet.id = ?", subnetID).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a subnet with id %d", subnetID)
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
		Join("INNER JOIN app_to_subnet AS atos ON atos.local_subnet_id = ? AND atos.app_id = ?", localSubnetID, appID).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("Apps")

	// Optionally filter by IPv4 or IPv6 subnets.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting subnets by local subnet id %d and app id %d", localSubnetID, appID)
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
		Relation("Apps").
		Where("subnet.prefix = ?", prefix).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting subnets with prefix %s", prefix)
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
		Relation("Apps").
		OrderExpr("id ASC")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting all subnets for family %d", family)
		return nil, err
	}
	return subnets, err
}

// Associates an application with the subnet having a specified ID and prefix.
func AddAppToSubnet(db *pg.DB, subnet *Subnet, app *App) error {
	localSubnetID := int64(0)
	// If the prefix is available we should try to match the subnet prefix
	// with the app's configuration and retrieve the local subnet id from
	// there.
	if len(subnet.Prefix) > 0 {
		localSubnetID = app.GetLocalSubnetID(subnet.Prefix)
	}
	assoc := AppToSubnet{
		AppID:         app.ID,
		SubnetID:      subnet.ID,
		LocalSubnetID: localSubnetID,
	}
	// Try to insert. If such association already exists we could maybe do
	// nothing, but we do update instead to force setting the new value
	// of the local_subnet_id if it has changed.
	_, err := db.Model(&assoc).
		Column("app_id").
		Column("subnet_id").
		Column("local_subnet_id").
		OnConflict("(app_id, subnet_id) DO UPDATE").
		Set("local_subnet_id = EXCLUDED.local_subnet_id").
		Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with associating the app with id %d with the subnet with id %d",
			app.ID, subnet.ID)
	}
	return err
}

// Dissociates an application from the subnet having a specified id.
// The first returned value indicates if any row was removed from the
// app_to_service table.
func DeleteAppFromSubnet(db *pg.DB, subnetID int64, appID int64) (bool, error) {
	assoc := &AppToSubnet{
		AppID:    appID,
		SubnetID: subnetID,
	}
	rows, err := db.Model(assoc).WherePK().Delete()
	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with deleting an app with id %d from the subnet with %d",
			appID, subnetID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Finds and returns an app associated with a subnet having the specified id.
func (s *Subnet) GetApp(appID int64) *SubnetAttachedApp {
	for _, a := range s.Apps {
		app := a
		if a.ID == appID {
			return app
		}
	}
	return nil
}
