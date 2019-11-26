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


// Part of machine table in database that describes state of machine. In DB it is stored as JSONB.
type MachineState struct {
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
}

// Represents a machine held in machine table in the database.
type Machine struct {
	Id          int64
	Created     time.Time
	Deleted     time.Time
	Address     string
	LastVisited time.Time
	Error       string
	State       MachineState
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

func GetMachines(db *pg.DB, offset int64, limit int64, text string) ([]Machine, int64, error) {
	if limit == 0 {
		limit = 10
	}
	var machines []Machine

	// prepare query
	q := db.Model(&machines).Where("deleted is NULL")
	if text != "" {
		text = "%" + text + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("address ILIKE ?", text).
				WhereOr("state->>'agent_version' ILIKE ?", text).
				WhereOr("state->>'hostname' ILIKE ?", text).
				WhereOr("state->>'os' ILIKE ?", text).
				WhereOr("state->>'platform' ILIKE ?", text).
				WhereOr("state->>'platform_family' ILIKE ?", text).
				WhereOr("state->>'platform_version' ILIKE ?", text).
				WhereOr("state->>'kernel_version' ILIKE ?", text).
				WhereOr("state->>'kernel_arch' ILIKE ?", text).
				WhereOr("state->>'virtualization_system' ILIKE ?", text).
				WhereOr("state->>'virtualization_role' ILIKE ?", text).
				WhereOr("state->>'host_id' ILIKE ?", text)
			return qq, nil
		})
	}

	// and then, first get total count
	total, err := q.Clone().Count()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting machines total")
	}

	// then retrive given page of rows
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting machines")
	}
	return machines, int64(total), nil
}

func DeleteMachine(db *pg.DB, machine *Machine) error {
	machine.Deleted = stork.UTCNow()
	err := db.Update(machine)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting machine %v", machine.Id)
	}
	return nil
}
