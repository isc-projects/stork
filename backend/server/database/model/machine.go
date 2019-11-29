package dbmodel

import (
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
	AgentPort   int64
	LastVisited time.Time
	Error       string
	State       MachineState
	Services    []Service
}

func AddMachine(db *pg.DB, machine *Machine) error {
	log.Infof("inserting machine %+v", machine)
	err := db.Insert(machine)
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting machine %+v", machine)
	}
	return err
}

func GetMachineByAddressAndAgentPort(db *pg.DB, address string, agentPort int64, withDeleted bool) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine)
	q = q.Where("address = ?", address)
	q = q.Where("agent_port = ?", agentPort)
	if !withDeleted {
		q = q.Where("deleted is NULL")
	}
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %s:%d", address, agentPort)
	}
	return &machine, nil
}

func GetMachineById(db *pg.DB, id int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("machine.id = ?", id)
	q = q.Relation("Services")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %v", id)
	}
	return &machine, nil
}

func RefreshMachineFromDb(db *pg.DB, machine *Machine) error {
	machine.Services = []Service{}
	q := db.Model(machine).Where("id = ?", machine.Id)
	q = q.Relation("Services")
	err := q.Select()
	if err != nil {
		return errors.Wrapf(err, "problem with getting machine %v", machine.Id)
	}
	return nil
}

// Fetches a collection of services from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned.
func GetMachinesByPage(db *pg.DB, offset int64, limit int64, text string) ([]Machine, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var machines []Machine

	// prepare query
	q := db.Model(&machines).Where("deleted is NULL")
	q = q.Relation("Services")
	if text != "" {
		text = "%" + text + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("address ILIKE ?", text).
				WhereOr("state->>'AgentVersion' ILIKE ?", text).
				WhereOr("state->>'Hostname' ILIKE ?", text).
				WhereOr("state->>'Os' ILIKE ?", text).
				WhereOr("state->>'Platform' ILIKE ?", text).
				WhereOr("state->>'PlatformFamily' ILIKE ?", text).
				WhereOr("state->>'PlatformVersion' ILIKE ?", text).
				WhereOr("state->>'KernelVersion' ILIKE ?", text).
				WhereOr("state->>'KernelArch' ILIKE ?", text).
				WhereOr("state->>'VirtualizationSystem' ILIKE ?", text).
				WhereOr("state->>'VirtualizationRole' ILIKE ?", text).
				WhereOr("state->>'HostID' ILIKE ?", text)
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
