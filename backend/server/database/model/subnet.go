package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"

	"time"
)

type Subnet struct {
	ID      int64
	Created time.Time
	Prefix  string

	SharedNetworkID int64
	SharedNetwork   *SharedNetwork

	AddressPools []AddressPool
	PrefixPools  []PrefixPool
}

func addSubnetPools(dbIface interface{}, subnet *Subnet) (err error) {
	if len(subnet.AddressPools) == 0 && len(subnet.PrefixPools) == 0 {
		return nil
	}

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

	for _, p := range subnet.AddressPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
	}
	for _, p := range subnet.PrefixPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
	}

	if db != nil {
		err = tx.Commit()
		if err != nil {
			err = errors.Wrapf(err, "problem with committing pools into a subnet with id %d", subnet.ID)
		}
	}

	return err
}

func addSubnetWithPools(tx *pg.Tx, subnet *Subnet) error {
	_, err := tx.Model(subnet).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new subnet with prefix %s", subnet.Prefix)
		return err
	}

	err = addSubnetPools(tx, subnet)
	if err != nil {
		return err
	}
	return err
}

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

func GetSubnetByPrefix(db *pg.DB, prefix string) (*Subnet, error) {
	subnet := &Subnet{}
	err := db.Model(subnet).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Where("subnet.prefix = ?", prefix).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a subnet with prefix %s", prefix)
		return nil, err
	}
	return subnet, err
}
