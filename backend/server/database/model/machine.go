package dbmodel

import (
	//	"fmt"
	"time"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork"
)

// Represents a user held in system_user table in the database.
type Machine struct {
	Id                   int
	Created              time.Time
	Deleted              time.Time
	Address              string
	AgentVersion         string
	Cpus                 int64
	CpusLoad             string
	Memory               int64
	Hostname             string
	Uptime               int64
	UsedMemory           int64
	Os                   string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	KernelVersion        string
	KernelArch           string
	VirtualizationSystem string
	VirtualizationRole   string
	HostID               string
	LastVisited          time.Time
	Error                string
}

func AddMachine(db *pg.DB, machine *Machine) error {
	log.Infof("inserting machine %+v", machine)
	err := db.Insert(machine)
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting machine %v", machine.Address)
	}
	return err
}

func GetMachineByAddress(db *pg.DB, address string, withDeleted bool) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("address = ?", address)
	if !withDeleted {
		q = q.Where("deleted is NULL")
	}
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %v", address)
	}
	return &machine, nil
}

func GetMachineById(db *pg.DB, id int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("id = ?", id)
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %v", id)
	}
	return &machine, nil
}

func GetMachines(db *pg.DB, offset int64, limit int64, text string) ([]Machine, int, error) {
	if limit == 0 {
		limit = 10
	}
	var machines []Machine

	total, err := db.Model(&machines).Count()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting machines total")
	}

	q := db.Model(&machines).Where("deleted is NULL")
	if text != "" {
		text = "%" + text + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("address LIKE ?", text).
				WhereOr("agent_version LIKE ?", text).
				WhereOr("hostname LIKE ?", text).
				WhereOr("os LIKE ?", text).
				WhereOr("platform LIKE ?", text).
				WhereOr("platform_family LIKE ?", text).
				WhereOr("platform_version LIKE ?", text).
				WhereOr("kernel_version LIKE ?", text).
				WhereOr("kernel_arch LIKE ?", text).
				WhereOr("virtualization_system LIKE ?", text).
				WhereOr("virtualization_role LIKE ?", text).
				WhereOr("host_id LIKE ?", text)
			return qq, nil
		})
	}
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting machines")
	}
	return machines, total, nil
}

func DeleteMachine(db *pg.DB, machine *Machine) error {
	machine.Deleted = stork.UTCNow()
	err := db.Update(machine)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting machine %v", machine.Id)
	}
	return nil
}
