package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	ID            int64
	CreatedAt     time.Time
	Address       string
	AgentPort     int64
	LastVisitedAt time.Time
	Error         string
	State         MachineState
	Apps          []*App
}

func AddMachine(db *pg.DB, machine *Machine) error {
	log.Infof("inserting machine %+v", machine)
	err := db.Insert(machine)
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting machine %+v", machine)
	}
	return err
}

func GetMachineByAddressAndAgentPort(db *pg.DB, address string, agentPort int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine)
	q = q.Where("address = ?", address)
	q = q.Where("agent_port = ?", agentPort)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %s:%d", address, agentPort)
	}
	return &machine, nil
}

func GetMachineByID(db *pg.DB, id int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("machine.id = ?", id)
	q = q.Relation("Apps.Daemons")
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting machine %v", id)
	}

	return &machine, nil
}

func RefreshMachineFromDb(db *pg.DB, machine *Machine) error {
	machine.Apps = []*App{}
	q := db.Model(machine).Where("id = ?", machine.ID)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if err != nil {
		return errors.Wrapf(err, "problem with getting machine %v", machine.ID)
	}

	return nil
}

// Fetches a collection of machines from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. Limit has to be greater then 0, otherwise error is
// returned. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting.  in SortDirAny is used then ASC
// order is used.
func GetMachinesByPage(db *pg.DB, offset int64, limit int64, filterText *string, sortField string, sortDir SortDirEnum) ([]Machine, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	q = q.Relation("Apps.AccessPoints")
	if filterText != nil {
		text := "%" + *filterText + "%"
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

	// prepare sorting expression, offser and limit
	ordExpr := prepareOrderExpr("machine", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting machines")
	}

	return machines, int64(total), nil
}

func DeleteMachine(db *pg.DB, machine *Machine) error {
	err := db.Delete(machine)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting machine %v", machine.ID)
	}
	return nil
}
