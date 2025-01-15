package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
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
	ID              int64
	CreatedAt       time.Time
	Address         string
	AgentPort       int64
	LastVisitedAt   time.Time
	Error           string
	State           MachineState
	Apps            []*App `pg:"rel:has-many"`
	AgentToken      string
	CertFingerprint [32]byte
	Authorized      bool `pg:",use_zero"`
}

// Identifier of the relations between the machine and other tables.
type MachineRelation string

// Names of the machine table relations. They must be valid in the go-pg sense.
const (
	MachineRelationApps             MachineRelation = "Apps"
	MachineRelationDaemons          MachineRelation = "Apps.Daemons"
	MachineRelationKeaDaemons       MachineRelation = "Apps.Daemons.KeaDaemon"
	MachineRelationBind9Daemons     MachineRelation = "Apps.Daemons.Bind9Daemon"
	MachineRelationDaemonLogTargets MachineRelation = "Apps.Daemons.LogTargets"
	MachineRelationAppAccessPoints  MachineRelation = "Apps.AccessPoints"
	MachineRelationKeaDHCPConfigs   MachineRelation = "Apps.Daemons.KeaDaemon.KeaDHCPDaemon"
	MachineRelationDaemonHAServices MachineRelation = "Apps.Daemons.Services.HAService"
)

// MachineTag is an interface implemented by the dbmodel.Machine exposing functions
// to create events referencing machines.
type MachineTag interface {
	GetID() int64
	GetAddress() string
	GetAgentPort() int64
	GetHostname() string
}

// Add new machine to database.
func AddMachine(db pg.DBI, machine *Machine) error {
	_, err := db.Model(machine).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem inserting machine %+v", machine)
	}
	return err
}

// Update a machine in database.
func UpdateMachine(db *pg.DB, machine *Machine) error {
	result, err := db.Model(machine).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating machine %+v", machine)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "machine with ID %d does not exist", machine.ID)
	}
	return err
}

// Get a machine by address and agent port.
func GetMachineByAddressAndAgentPort(db *pg.DB, address string, agentPort int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine)
	q = q.Where("address = ?", address)
	q = q.Where("agent_port = ?", agentPort)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting machine %s:%d", address, agentPort)
	}
	return &machine, nil
}

// Get a machine by the machine address and the access point port.
// Optionally, it filters access points by type.
func GetMachineByAddressAndAccessPointPort(db *pg.DB, machineAddress string, accessPointPort int64, accessPointType *string) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).
		Relation(string(MachineRelationAppAccessPoints)).
		Relation(string(MachineRelationDaemons)).
		Join("JOIN access_point").JoinOn("machine.id = access_point.machine_id").
		Where("machine.address = ?", machineAddress).
		Where("access_point.port = ?", accessPointPort)

	if accessPointType != nil {
		q = q.Where("access_point.type = ?", accessPointType)
	}

	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting machine by the '%s' machine address and the '%d' access point port", machineAddress, accessPointPort)
	}
	return &machine, nil
}

// Get a machine by its ID with default relations.
func GetMachineByID(db *pg.DB, id int64) (*Machine, error) {
	return GetMachineByIDWithRelations(db, id,
		MachineRelationAppAccessPoints,
		MachineRelationBind9Daemons,
		MachineRelationKeaDHCPConfigs)
}

// Get a machine by its ID with relations.
func GetMachineByIDWithRelations(db *pg.DB, id int64, relations ...MachineRelation) (*Machine, error) {
	tables := make([]string, len(relations))
	for idx, tableName := range relations {
		tables[idx] = string(tableName)
	}
	return getMachineByID(db, id, tables)
}

// Get a machine by its ID with relations - internal.
func getMachineByID(db *pg.DB, id int64, relations []string) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("machine.id = ?", id)
	for _, relation := range relations {
		q = q.Relation(relation)
	}
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting machine %v", id)
	}

	return &machine, nil
}

// Refresh machine from database.
func RefreshMachineFromDB(db *pg.DB, machine *Machine) error {
	machine.Apps = []*App{}
	q := db.Model(machine).Where("id = ?", machine.ID)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem getting machine %v", machine.ID)
	}

	return nil
}

// Fetches a collection of machines from the database.
//
// The offset and limit specify the beginning of the page and the
// maximum size of the page. Limit has to be greater then 0, otherwise
// error is returned.
//
// filterText allows filtering machines by provided text. It is check
// against several different fields in Machine record. If not provided
// then no filtering by text happens.
//
// authorized allows filtering machines by authorized field in Machine
// record. It can be true or false then authorized or unauthorized
// machines are returned. If it is nil then no filtering by authorized
// happens (ie. all machines are returned).
//
// sortField allows indicating sort column in database and sortDir
// allows selection the order of sorting. If sortField is empty then
// id is used for sorting.  in SortDirAny is used then ASC order is
// used.
func GetMachinesByPage(db *pg.DB, offset int64, limit int64, filterText *string, authorized *bool, sortField string, sortDir SortDirEnum) ([]Machine, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	q = q.Relation("Apps.AccessPoints")
	q = q.Relation("Apps.Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Apps.Daemons.Bind9Daemon")

	// prepare filtering by text
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

	// prepare filtering by authorized
	if authorized != nil {
		q = q.Where("authorized = ?", *authorized)
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("machine", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []Machine{}, 0, nil
		}
		return nil, 0, pkgerrors.Wrapf(err, "problem getting machines")
	}

	return machines, int64(total), nil
}

// Get all machines from database. It can be filtered by authorized field.
func GetAllMachines(db *pg.DB, authorized *bool) ([]Machine, error) {
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	if authorized != nil {
		q = q.Where("authorized = ?", *authorized)
	}
	q = q.Relation("Apps.AccessPoints")
	q = q.Relation("Apps.Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Apps.Daemons.Bind9Daemon")
	q = q.Relation("Apps.Daemons.ConfigReview")

	err := q.Select()
	if err != nil && errors.Is(err, pg.ErrNoRows) {
		return nil, pkgerrors.Wrapf(err, "problem getting machines")
	}

	return machines, nil
}

// Get all machines from database with minimal data about Kea daemons. It can be filtered by authorized field.
func GetAllMachinesSimplified(db *pg.DB, authorized *bool) ([]Machine, error) {
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	if authorized != nil {
		q = q.Where("authorized = ?", *authorized)
	}
	q = q.Relation("Apps.Daemons")

	err := q.Select()
	if err != nil && errors.Is(err, pg.ErrNoRows) {
		return nil, pkgerrors.Wrapf(err, "problem getting machines")
	}

	return machines, nil
}

// Returns the number of unauthorized machines.
func GetUnauthorizedMachinesCount(db *pg.DB) (int, error) {
	count, err := db.Model((*Machine)(nil)).Where("authorized = ?", false).Count()
	return count, pkgerrors.Wrapf(err, "problem counting unauthorized machines")
}

// Delete a machine from database.
func DeleteMachine(db *pg.DB, machine *Machine) error {
	result, err := db.Model(machine).WherePK().Delete()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem deleting machine %v", machine.ID)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "machine with ID %d does not exist", machine.ID)
	}
	return nil
}

// MachineTag interface implementation.

// Returns machine ID.
func (machine *Machine) GetID() int64 {
	return machine.ID
}

// Returns machine address.
func (machine *Machine) GetAddress() string {
	return machine.Address
}

// Returns machine agent port.
func (machine *Machine) GetAgentPort() int64 {
	return machine.AgentPort
}

// Returns hostname.
func (machine *Machine) GetHostname() string {
	return machine.State.Hostname
}
