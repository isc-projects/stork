package dbmodel

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
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
	Daemons         []*Daemon `pg:"rel:has-many"`
	AgentToken      string
	CertFingerprint [32]byte
	Authorized      bool `pg:",use_zero"`
}

// Identifier of the relations between the machine and other tables.
type MachineRelation string

// Names of the machine table relations. They must be valid in the go-pg sense.
const (
	MachineRelationDaemons            MachineRelation = "Daemons"
	MachineRelationKeaDaemons         MachineRelation = "Daemons.KeaDaemon"
	MachineRelationBind9Daemons       MachineRelation = "Daemons.Bind9Daemon"
	MachineRelationPDNSDaemons        MachineRelation = "Daemons.PDNSDaemon"
	MachineRelationDaemonLogTargets   MachineRelation = "Daemons.LogTargets"
	MachineRelationDaemonAccessPoints MachineRelation = "Daemons.AccessPoints"
	MachineRelationKeaDHCPConfigs     MachineRelation = "Daemons.KeaDaemon.KeaDHCPDaemon"
	MachineRelationDaemonHAServices   MachineRelation = "Daemons.Services.HAService"
	MachineRelationDaemonConfigReview MachineRelation = "Daemons.ConfigReview"
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
	q = q.Relation(string(MachineRelationDaemonAccessPoints))
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting machine %s:%d", address, agentPort)
	}

	for _, daemon := range machine.Daemons {
		daemon.Machine = &machine
	}

	return &machine, nil
}

// Get a machine by the machine address and the access point port.
// Optionally, it filters access points by type.
func GetMachineByAddressAndAccessPointPort(db *pg.DB, machineAddress string, accessPointPort int64, accessPointType *AccessPointType) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).
		Relation(string(MachineRelationDaemonAccessPoints)).
		Join("JOIN daemon").JoinOn("machine.id = daemon.machine_id").
		Join("JOIN access_point").JoinOn("daemon.id = access_point.daemon_id").
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

	for _, daemon := range machine.Daemons {
		daemon.Machine = &machine
	}

	return &machine, nil
}

// Get a machine by its ID with default relations.
func GetMachineByID(db *pg.DB, id int64) (*Machine, error) {
	return GetMachineByIDWithRelations(db, id,
		MachineRelationDaemonLogTargets,
		MachineRelationDaemonAccessPoints,
		MachineRelationBind9Daemons,
		MachineRelationKeaDHCPConfigs,
		MachineRelationPDNSDaemons)
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

	for _, daemon := range machine.Daemons {
		daemon.Machine = &machine
	}

	return &machine, nil
}

// Refresh machine from database.
func RefreshMachineFromDB(db *pg.DB, machine *Machine) error {
	machine.Daemons = []*Daemon{}
	q := db.Model(machine).Where("id = ?", machine.ID)
	q = q.Relation(string(MachineRelationDaemonAccessPoints))
	err := q.Select()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem getting machine %v", machine.ID)
	}

	for _, daemon := range machine.Daemons {
		daemon.Machine = machine
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
	q = q.Relation(string(MachineRelationDaemonAccessPoints))
	q = q.Relation(string(MachineRelationDaemonLogTargets))
	q = q.Relation(string(MachineRelationKeaDHCPConfigs))
	q = q.Relation(string(MachineRelationBind9Daemons))
	q = q.Relation(string(MachineRelationPDNSDaemons))

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

	// REST API is accepting simplified sortField names. Convert it to appropriate field names accepted by DB.
	var dbSortField string
	switch sortField {
	case "hostname":
		dbSortField = "state->'Hostname'"
	case "cpus":
		dbSortField = "state->'Cpus'"
	case "cpus_load":
		dbSortField = "state->'CpusLoad'"
	case "memory":
		dbSortField = "state->'Memory'"
	case "used_memory":
		dbSortField = "state->'UsedMemory'"
	case "agent_version":
		dbSortField = "state->'AgentVersion'"
	default:
		dbSortField = sortField
	}

	// prepare sorting expression, offset and limit
	ordExpr, _ := prepareOrderAndDistinctExpr("machine", dbSortField, sortDir, nil)
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

	for _, machine := range machines {
		for _, daemon := range machine.Daemons {
			daemon.Machine = &machine
		}
	}

	return machines, int64(total), nil
}

// Get all machines from database. It can be filtered by authorized field.
func GetAllMachines(db *pg.DB, authorized *bool) ([]Machine, error) {
	return GetAllMachinesWithRelations(db, authorized,
		MachineRelationDaemonLogTargets,
		MachineRelationDaemonAccessPoints,
		MachineRelationKeaDHCPConfigs,
		MachineRelationBind9Daemons,
		MachineRelationPDNSDaemons,
		MachineRelationDaemonConfigReview)
}

// Get all machines from database with specific relations. It can be filtered
// by authorized field.
func GetAllMachinesWithRelations(db *pg.DB, authorized *bool, relations ...MachineRelation) ([]Machine, error) {
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	if authorized != nil {
		q = q.Where("authorized = ?", *authorized)
	}
	for _, relation := range relations {
		// The daemons must be returned in determined order because some
		// unit tests are relying on it. The order is determined by the ID of
		// the daemon. It is a primary key. It is default order on most
		// PostgreSQL versions, but some (older?) distributions don't guarantee
		// it. To be sure that the order is always the same, we need to specify
		// it explicitly.
		if strings.Contains(string(relation), "Daemons") {
			q = q.Relation(string(MachineRelationDaemons), func(q *orm.Query) (*orm.Query, error) {
				return q.Order("daemon.id ASC"), nil
			})
			if relation == MachineRelationDaemons {
				continue
			}
		}

		q = q.Relation(string(relation), func(q *orm.Query) (*orm.Query, error) {
			return q, nil
		})
	}

	err := q.Select()
	if err != nil && errors.Is(err, pg.ErrNoRows) {
		return nil, pkgerrors.Wrapf(err, "problem getting machines")
	}

	for _, machine := range machines {
		for _, daemon := range machine.Daemons {
			daemon.Machine = &machine
		}
	}

	return machines, nil
}

// Get all machines from database without involving any DB relations. This is to have as lightweight DB query as possible. It can be filtered by authorized field.
func GetAllMachinesNoRelations(db *pg.DB, authorized *bool) ([]Machine, error) {
	return GetAllMachinesWithRelations(db, authorized)
}

// Returns the number of unauthorized machines.
func GetUnauthorizedMachinesCount(db *pg.DB) (int, error) {
	count, err := db.Model((*Machine)(nil)).Where("authorized = ?", false).Count()
	return count, pkgerrors.Wrapf(err, "problem counting unauthorized machines")
}

// Delete a machine from database. The machine must include non-nil Daemons
// field (though it may be an empty slice). The Daemons field is used to
// delete orphaned objects (e.g., subnets, zones) after the machine is deleted.
// The whole operation is transactional, so it is rolled back if it fails at
// any stage.
func DeleteMachine(db *pg.DB, machine *Machine) error {
	if machine.Daemons == nil {
		return pkgerrors.Errorf("deleted machine with ID %d has no daemons relation", machine.ID)
	}
	return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		result, err := db.Model(machine).WherePK().Delete()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem deleting machine %v", machine.ID)
		} else if result.RowsAffected() <= 0 {
			return pkgerrors.Wrapf(ErrNotExists, "machine with ID %d does not exist", machine.ID)
		}
		// Deleting the machine may leave some orphaned objects behind.
		// Let's make sure they are deleted.
		daemonNames := make(map[daemonname.Name]bool)
		fns := []func(tx dbops.DBI) (int64, error){}
		for _, daemon := range machine.Daemons {
			daemonNames[daemon.Name] = true
		}
		for daemonName := range daemonNames {
			switch daemonName {
			case daemonname.Bind9, daemonname.PDNS:
				fns = append(fns, DeleteOrphanedZones)
			case daemonname.DHCPv4, daemonname.DHCPv6:
				fns = append(fns, DeleteOrphanedSubnets, DeleteOrphanedHosts, DeleteOrphanedSharedNetworks)
			case daemonname.CA, daemonname.D2, daemonname.NetConf:
				// No orphaned objects to delete.
			}
		}
		for _, fn := range fns {
			if _, err := fn(tx); err != nil {
				return err
			}
		}
		return nil
	})
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

// Return the label of the daemon for identification in the UI and logs.
func (machine *Machine) GetLabel() string {
	if machine.State.Hostname != "" {
		return machine.State.Hostname
	}
	return machine.Address
}
