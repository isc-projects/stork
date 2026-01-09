package dbmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	errors "github.com/pkg/errors"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/daemondata/bind9stats"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
)

// Available daemon relations to other tables.
type DaemonRelation = string

const (
	DaemonRelationAccessPoints  DaemonRelation = "AccessPoints"
	DaemonRelationLogTargets    DaemonRelation = "LogTargets"
	DaemonRelationKeaDaemon     DaemonRelation = "KeaDaemon"
	DaemonRelationKeaDHCPDaemon DaemonRelation = "KeaDaemon.KeaDHCPDaemon"
	DaemonRelationBind9Daemon   DaemonRelation = "Bind9Daemon"
	DaemonRelationPDNSDaemon    DaemonRelation = "PDNSDaemon"
	DaemonRelationMachine       DaemonRelation = "Machine"
	DaemonRelationConfigReview  DaemonRelation = "ConfigReview"
	DaemonRelationServices      DaemonRelation = "Services"
	DaemonRelationHAService     DaemonRelation = "Services.HAService"
)

// KEA

// A structure reflecting Kea DHCP stats for daemon. It is stored
// as a JSONB value in SQL and unmarshaled in this structure.
type KeaDHCPDaemonStats struct {
	RPS1 float32 `pg:"rps1"`
	RPS2 float32 `pg:"rps2"`
}

// A structure holding Kea DHCP specific information about a daemon. It
// reflects the kea_dhcp_daemon table which extends the daemon and
// kea_daemon tables with the Kea DHCPv4 or DHCPv6 specific information.
type KeaDHCPDaemon struct {
	tableName   struct{} `pg:"kea_dhcp_daemon"` //nolint:unused
	ID          int64
	KeaDaemonID int64
	Stats       KeaDHCPDaemonStats
}

// A structure holding common information for all Kea daemons. It
// reflects the information stored in the kea_daemon table.
type KeaDaemon struct {
	ID         int64
	Config     *KeaConfig `pg:",use_zero"`
	ConfigHash string
	DaemonID   int64

	KeaDHCPDaemon *KeaDHCPDaemon `pg:"rel:belongs-to"`
}

// BIND 9

// A structure reflecting BIND 9 stats for a daemon. It is stored as a JSONB
// value in SQL and unmarshaled to this structure.
type Bind9DaemonStats struct {
	ZoneCount          int64
	AutomaticZoneCount int64
	NamedStats         bind9stats.Bind9NamedStats
}

// A structure holding BIND9 daemon specific information.
type Bind9Daemon struct {
	ID       int64
	DaemonID int64
	Stats    Bind9DaemonStats
}

// A structure holding PowerDNS daemon specific information.
type PDNSDaemonDetails struct {
	URL              string
	ConfigURL        string
	ZonesURL         string
	AutoprimariesURL string
}

// A structure holding PowerDNS daemon specific information.
type PDNSDaemon struct {
	tableName struct{} `pg:"pdns_daemon"` //nolint:unused
	ID        int64
	DaemonID  int64
	Details   PDNSDaemonDetails
}

// A structure reflecting all SQL tables holding information about the
// daemons of various types. It embeds the KeaDaemon structure which
// holds Kea DHCP specific information for Kea daemons. It is nil
// if the daemon is not of the Kea type. Similarly, it holds BIND9
// specific information in the Bind9Daemon structure if the daemon
// type is BIND9. The daemon structure is to be extended with additional
// embedded structures as more daemon types are defined.
type Daemon struct {
	ID              int64
	Pid             int32
	Name            daemonname.Name
	Active          bool `pg:",use_zero"`
	Monitored       bool `pg:",use_zero"`
	Version         string
	ExtendedVersion string
	Uptime          int64
	CreatedAt       time.Time
	ReloadedAt      time.Time

	MachineID int64
	Machine   *Machine `pg:"rel:has-one"`

	Services []*Service `pg:"many2many:daemon_to_service,fk:daemon_id,join_fk:service_id"`

	AccessPoints []*AccessPoint `pg:"rel:has-many"`
	LogTargets   []*LogTarget   `pg:"rel:has-many"`

	KeaDaemon   *KeaDaemon   `pg:"rel:belongs-to"`
	Bind9Daemon *Bind9Daemon `pg:"rel:belongs-to"`
	PDNSDaemon  *PDNSDaemon  `pg:"rel:belongs-to"`

	ConfigReview *ConfigReview `pg:"rel:belongs-to"`
}

// GetAccessPoint returns the access point of the given access point type.
func (d Daemon) GetAccessPoint(accessPointType AccessPointType) (ap *AccessPoint, err error) {
	for _, point := range d.AccessPoints {
		if point.Type == accessPointType {
			return point, nil
		}
	}
	return nil, errors.Errorf("no access point of type %s found for daemon ID %d", accessPointType, d.ID)
}

// Returns MachineTag interface to the machine owning the daemon.
func (d Daemon) GetMachineTag() MachineTag {
	return d.Machine
}

// Return the label of the daemon for identification in the UI and logs.
func (d Daemon) GetLabel() string {
	formattedDaemonName := string(d.Name)
	switch d.Name {
	case daemonname.CA:
		formattedDaemonName = "CA"
	case daemonname.D2:
		formattedDaemonName = "DDNS"
	case daemonname.NetConf:
		formattedDaemonName = "NetConf"
	case daemonname.DHCPv4:
		formattedDaemonName = "DHCPv4"
	case daemonname.DHCPv6:
		formattedDaemonName = "DHCPv6"
	case daemonname.Bind9:
		formattedDaemonName = "BIND9"
	case daemonname.PDNS:
		formattedDaemonName = "PowerDNS"
	}

	if d.Machine != nil {
		machineLabel := d.Machine.GetLabel()
		return fmt.Sprintf("%s@%s", formattedDaemonName, machineLabel)
	}

	accessPoint, err := d.GetAccessPoint(AccessPointControl)
	if err == nil {
		return fmt.Sprintf("%s@%s", formattedDaemonName, accessPoint.Address)
	}
	return formattedDaemonName
}

// Structure representing HA service information displayed for the daemon
// in the dashboard.
type DaemonServiceOverview struct {
	State         string
	LastFailureAt time.Time
}

// DaemonTag is an interface implemented by the dbmodel.Daemon exposing functions
// to create events referencing machines.
type DaemonTag interface {
	GetID() int64
	GetName() daemonname.Name
	GetMachineID() int64
}

// Creates an instance of a daemon with its references initialized to empty
// structures.
func NewDaemon(machine *Machine, name daemonname.Name, active bool, accessPoints []*AccessPoint) *Daemon {
	daemon := &Daemon{
		Name:         name,
		Active:       active,
		Monitored:    true,
		MachineID:    machine.ID,
		Machine:      machine,
		AccessPoints: accessPoints,
	}

	switch name {
	case daemonname.CA, daemonname.D2, daemonname.NetConf:
		daemon.KeaDaemon = &KeaDaemon{}
	case daemonname.DHCPv4, daemonname.DHCPv6:
		daemon.KeaDaemon = &KeaDaemon{KeaDHCPDaemon: &KeaDHCPDaemon{}}
	case daemonname.Bind9:
		daemon.Bind9Daemon = &Bind9Daemon{}
	case daemonname.PDNS:
		daemon.PDNSDaemon = &PDNSDaemon{}
	}

	return daemon
}

// Gets daemon by ID with relations.
func GetDaemonByIDWithRelations(dbi pg.DBI, id int64, relations ...DaemonRelation) (*Daemon, error) {
	daemon := Daemon{}
	q := dbi.Model(&daemon)
	for _, relation := range relations {
		q = q.Relation(relation)
	}
	q = q.Where("daemon.id = ?", id)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem getting daemon %v", id)
	}
	return &daemon, nil
}

// Gets daemon by ID with default relations.
func GetDaemonByID(dbi pg.DBI, id int64) (*Daemon, error) {
	return GetDaemonByIDWithRelations(
		dbi, id,
		DaemonRelationLogTargets,
		DaemonRelationAccessPoints, DaemonRelationMachine,
		DaemonRelationKeaDHCPDaemon, DaemonRelationBind9Daemon,
		DaemonRelationPDNSDaemon,
	)
}

// Gets Kea daemon by ID.
// It doesn't validate that the daemon is indeed a Kea daemon.
func GetKeaDaemonByID(dbi pg.DBI, id int64) (*Daemon, error) {
	return GetDaemonByIDWithRelations(
		dbi, id,
		DaemonRelationLogTargets,
		DaemonRelationAccessPoints, DaemonRelationMachine,
		DaemonRelationKeaDHCPDaemon,
	)
}

// Get DNS daemon by ID.
// It doesn't validate that the daemon is indeed a DNS daemon.
func GetDNSDaemonByID(dbi pg.DBI, id int64) (*Daemon, error) {
	return GetDaemonByIDWithRelations(
		dbi, id,
		DaemonRelationAccessPoints, DaemonRelationMachine,
		DaemonRelationBind9Daemon, DaemonRelationPDNSDaemon,
	)
}

// Get selected Kea daemons by their ids.
// It doesn't validate that the daemons are indeed Kea daemons.
func GetKeaDaemonsByIDs(dbi pg.DBI, ids []int64) (daemons []Daemon, err error) {
	err = dbi.Model(&daemons).
		Relation(DaemonRelationLogTargets).
		Relation(DaemonRelationAccessPoints).
		Relation(DaemonRelationMachine).
		Relation(DaemonRelationKeaDHCPDaemon).
		Where("daemon.id IN (?)", pg.In(ids)).
		OrderExpr("daemon.id ASC").
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		var sids []string
		for _, id := range ids {
			sids = append(sids, fmt.Sprintf("%d", id))
		}
		return nil, errors.Wrapf(err, "problem selecting daemons with IDs: %s",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Get daemons by their machine ID.
func GetDaemonsByMachine(dbi pg.DBI, machineID int64) (daemons []Daemon, err error) {
	err = dbi.Model(&daemons).
		Relation(DaemonRelationLogTargets).
		Relation(DaemonRelationAccessPoints).
		Relation(DaemonRelationMachine).
		Relation(DaemonRelationKeaDaemon).
		Relation(DaemonRelationBind9Daemon).
		Relation(DaemonRelationPDNSDaemon).
		Where("daemon.machine_id = ?", machineID).
		OrderExpr("daemon.id ASC").
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem selecting daemons for machine ID %d", machineID)
	}
	return daemons, nil
}

// Retrieves all daemons.
func GetAllDaemons(dbi dbops.DBI) ([]Daemon, error) {
	return GetAllDaemonsWithRelations(dbi,
		nil,
		nil,
		DaemonRelationAccessPoints,
		DaemonRelationMachine,
		DaemonRelationLogTargets,
		DaemonRelationKeaDHCPDaemon,
		DaemonRelationBind9Daemon,
		DaemonRelationPDNSDaemon,
	)
}

// Retrieves all daemons with provided relationships to other tables.
// It is possible to filter retrieved daemons by search text or dns/dhcp domain.
func GetAllDaemonsWithRelations(dbi dbops.DBI, filterText *string, filterDomain *string, relations ...DaemonRelation) ([]Daemon, error) {
	var daemons []Daemon

	q := dbi.Model(&daemons)
	for _, relation := range relations {
		q = q.Relation(relation)
	}
	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("name ILIKE ?", text)
			if slices.Contains(relations, DaemonRelationMachine) {
				qq = qq.WhereOr("machine.address ILIKE ?", text)
				qq = qq.WhereOr("machine.state->>'Hostname' ILIKE ?", text)
			}
			return qq, nil
		})
	}
	if filterDomain != nil {
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			var names []string
			switch *filterDomain {
			case "dns":
				names = []string{string(daemonname.Bind9), string(daemonname.PDNS)}
			case "dhcp":
				names = []string{
					string(daemonname.DHCPv4),
					string(daemonname.DHCPv6),
					string(daemonname.NetConf),
					string(daemonname.D2),
					string(daemonname.CA),
				}
			}
			qq = qq.Where("name IN (?)", pg.In(names))
			return qq, nil
		})
	}
	err := q.OrderExpr("id ASC").Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem getting daemons from the database")
	}
	return daemons, nil
}

// Fetches a collection of daemons from the database.
//
// The offset and limit specify the beginning of the page and the maximum size
// of the page. Limit has to be greater then 0, otherwise error is returned.
//
// filterText allows for filtering daemons by name, version, extended_version,
// machine address and hostname.
//
// daemonNames allows for filtering daemons by names. If no names are
// provided then daemons of all names are returned.
//
// sortField select sorting column in database and sortDir selects
// the sorting order. If sortField is empty then id is used for
// sorting.
//
// If SortDirAny is used then ASC order is used.
func GetDaemonsByPage(dbi dbops.DBI, offset int64, limit int64, filterText *string, sortField string, sortDir SortDirEnum, daemonNames ...daemonname.Name) ([]Daemon, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var daemons []Daemon

	// prepare query
	q := dbi.Model(&daemons).
		Relation(DaemonRelationAccessPoints).
		Relation(DaemonRelationMachine).
		Relation(DaemonRelationLogTargets)

	if len(daemonNames) == 0 {
		q = q.Relation(DaemonRelationHAService).
			Relation(DaemonRelationKeaDHCPDaemon).
			Relation(DaemonRelationBind9Daemon).
			Relation(DaemonRelationPDNSDaemon)
	} else {
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			for _, daemonName := range daemonNames {
				qq = qq.WhereOr("name = ?", daemonName)
				switch daemonName {
				case daemonname.DHCPv4, daemonname.DHCPv6:
					qq = qq.Relation(DaemonRelationHAService)
					qq = qq.Relation(DaemonRelationKeaDHCPDaemon)
				case daemonname.CA, daemonname.D2, daemonname.NetConf:
					qq = qq.Relation(DaemonRelationKeaDaemon)
				case daemonname.Bind9:
					qq = qq.Relation(DaemonRelationBind9Daemon)
				case daemonname.PDNS:
					qq = qq.Relation(DaemonRelationPDNSDaemon)
				}
			}
			return qq, nil
		})
	}

	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("name ILIKE ?", text)
			qq = qq.WhereOr("version ILIKE ?", text)
			qq = qq.WhereOr("extended_version ILIKE ?", text)
			qq = qq.WhereOr("machine.address ILIKE ?", text)
			qq = qq.WhereOr("machine.state->>'Hostname' ILIKE ?", text)
			return qq, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr, _ := prepareOrderAndDistinctExpr("daemon", sortField, sortDir, nil)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []Daemon{}, 0, nil
		}
		return nil, 0, errors.Wrapf(err, "problem getting daemons")
	}
	return daemons, int64(total), nil
}

// Get daemons by their name.
func GetDaemonsByName(dbi pg.DBI, names ...daemonname.Name) (daemons []Daemon, err error) {
	q := dbi.Model(&daemons).
		Relation(DaemonRelationAccessPoints).
		Relation(DaemonRelationMachine).
		Where("daemon.name IN (?)", pg.In(names)).
		OrderExpr("daemon.id ASC")

	for _, daemonName := range names {
		q = q.WhereOr("name = ?", daemonName)
		switch daemonName {
		case daemonname.DHCPv4, daemonname.DHCPv6:
			q = q.Relation(DaemonRelationHAService)
			q = q.Relation(DaemonRelationKeaDHCPDaemon)
		case daemonname.CA, daemonname.D2, daemonname.NetConf:
			q = q.Relation(DaemonRelationKeaDaemon)
		case daemonname.Bind9:
			q = q.Relation(DaemonRelationBind9Daemon)
		case daemonname.PDNS:
			q = q.Relation(DaemonRelationPDNSDaemon)
		}
	}

	if len(names) == 0 {
		q = q.Relation(DaemonRelationHAService).
			Relation(DaemonRelationKeaDHCPDaemon).
			Relation(DaemonRelationBind9Daemon).
			Relation(DaemonRelationPDNSDaemon)
	}

	err = q.Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem selecting daemons with names: %v", names)
	}
	return daemons, nil
}

// Get DHCP daemons (DHCPv4 and DHCPv6).
func GetDHCPDaemons(dbi pg.DBI) (daemons []Daemon, err error) {
	return GetDaemonsByName(dbi, daemonname.DHCPv4, daemonname.DHCPv6)
}

// Get DNS daemons (BIND9 and PowerDNS).
func GetDNSDaemons(dbi pg.DBI) (daemons []Daemon, err error) {
	return GetDaemonsByName(dbi, daemonname.Bind9, daemonname.PDNS)
}

// Get all Kea DHCP daemons.
func GetKeaDHCPDaemons(dbi pg.DBI) (daemons []Daemon, err error) {
	err = dbi.Model(&daemons).
		Relation(DaemonRelationLogTargets).
		Relation(DaemonRelationMachine).
		Relation(DaemonRelationKeaDHCPDaemon).
		Where("daemon.name ILIKE 'dhcp%'").
		OrderExpr("daemon.id ASC").
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		err = nil
	} else {
		err = errors.Wrapf(err, "problem with getting Kea DHCP daemons")
	}
	return
}

// Select one or more daemons for update. The main use case for this function is
// to prevent modifications and deletions of the daemons while the server inserts
// config reports for them. It must be called within a transaction and the selected
// rows remain locked until the transaction is committed or rolled back. Due to the
// limitations of the go-pg library, this function does not select all columns in the
// joined tables (machine and access points).
func GetDaemonsForUpdate(tx *pg.Tx, daemonsToSelect []*Daemon) ([]*Daemon, error) {
	var daemons []*Daemon

	// It is an error when no daemons are specified. Typically it will be one
	// daemon but there can be more.
	if len(daemonsToSelect) == 0 {
		return daemons, errors.New("no daemons specified for selection for update")
	}

	// Execute SELECT ... FROM daemon ... FOR UPDATE. It locks all selected
	// rows of the daemon table and the selected rows of the joined tables.
	// PostgreSQL does not allow for such locking when LEFT JOIN is used
	// because some joined rows may be NULL in this case. Unfortunately,
	// the go-pg Relation() does not allow for choosing between the INNER and
	// LEFT JOIN. Therefore, we can't use the Relation() call in this query.
	// Instead, we use explicit Join() and ColumnExpr() calls. It requires
	// explicitly specifying the selected column names. Thus, we limited the
	// selected columns to ID for joined tables to avoid having to maintain
	// the selected columns list. The query can be extended to select more
	// columns if necessary.
	var ids []int64
	for _, d := range daemonsToSelect {
		ids = append(ids, d.ID)
	}
	err := tx.Model(&daemons).
		ColumnExpr("daemon.*").
		Join("INNER JOIN machine ON daemon.machine_id = machine.id").
		Where("daemon.id IN (?)", pg.In(ids)).
		For("UPDATE").
		OrderExpr("daemon.id ASC").
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		var sids []string
		for _, id := range ids {
			sids = append(sids, fmt.Sprintf("%d", id))
		}
		return nil, errors.Wrapf(err, "problem selecting daemons with IDs: %s, for update",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Select one or more Kea daemons for update. The main use case for this function is
// to prevent modifications and deletions of the Kea daemons while the server inserts
// config reports for them. It must be called within a transaction and the selected
// rows remain locked until the transaction is committed or rolled back. Due to the
// limitations of the go-pg library, this function does not select all columns in the
// joined tables (machine and kea_daemon). It selects the config and the
// config_hash columns from the kea_daemon so the configreview implementation can
// verify that the configurations were not changed during the config review process.
// For other joined tables, this function merely returns id columns values.
func GetKeaDaemonsForUpdate(tx *pg.Tx, daemonsToSelect []*Daemon) ([]*Daemon, error) {
	var daemons []*Daemon

	// It is an error when no daemons are specified. Typically it will be one
	// daemon but there can be more.
	if len(daemonsToSelect) == 0 {
		return daemons, errors.New("no Kea daemons specified for selection for update")
	}

	// Execute SELECT ... FROM daemon ... FOR UPDATE. It locks all selected
	// rows of the daemon table and the selected rows of the joined tables.
	// PostgreSQL does not allow for such locking when LEFT JOIN is used
	// because some joined rows may be NULL in this case. Unfortunately,
	// the go-pg Relation() does not allow for choosing between the INNER and
	// LEFT JOIN. Therefore, we can't use the Relation() call in this query.
	// Instead, we use explicit Join() and ColumnExpr() calls. It requires
	// explicitly specifying the selected column names. Thus, we limited the
	// selected columns to ID for joined tables to avoid having to maintain
	// the selected columns list. The query can be extended to select more
	// columns if necessary.
	var ids []int64
	for _, d := range daemonsToSelect {
		ids = append(ids, d.ID)
	}
	err := tx.Model(&daemons).
		ColumnExpr("daemon.*").
		ColumnExpr("kea_daemon.id AS kea_daemon__id, kea_daemon.config AS kea_daemon__config, kea_daemon.config_hash AS kea_daemon__config_hash").
		ColumnExpr("machine.id AS daemon__machine__id").
		Join("INNER JOIN kea_daemon ON kea_daemon.daemon_id = daemon.id").
		Join("INNER JOIN machine ON machine.id = daemon.machine_id").
		Where("daemon.id IN (?)", pg.In(ids)).
		For("UPDATE").
		OrderExpr("daemon.id ASC").
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		var sids []string
		for _, id := range ids {
			sids = append(sids, fmt.Sprintf("%d", id))
		}
		return nil, errors.Wrapf(err, "problem selecting Kea daemons with IDs: %s, for update",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Adds a new daemon to the database.
func addDaemon(tx *pg.Tx, daemon *Daemon) error {
	if daemon.MachineID == 0 {
		return errors.New("daemon must have a machine ID set")
	}
	_, err := tx.Model(daemon).Insert()
	if err != nil {
		return errors.Wrapf(err, "problem adding daemon %v", daemon)
	}
	// Add access points.
	for _, accessPoint := range daemon.AccessPoints {
		accessPoint.DaemonID = daemon.ID
		err = addOrUpdateAccessPoint(tx, accessPoint)
		if err != nil {
			return errors.WithMessagef(err, "problem adding access point %v for daemon %d",
				accessPoint, daemon.ID)
		}
	}

	// Add references.
	switch {
	case daemon.KeaDaemon != nil:
		// Make sure that the kea_daemon references the daemon.
		daemon.KeaDaemon.DaemonID = daemon.ID
		err = upsertInTransaction(tx, daemon.KeaDaemon.ID, daemon.KeaDaemon)
		if err != nil {
			return errors.WithMessagef(err, "problem upserting Kea daemon %d: %v",
				daemon.ID, daemon.KeaDaemon)
		}

		if daemon.KeaDaemon.KeaDHCPDaemon != nil {
			// Make sure that the kea_dhcp_daemon references the kea_daemon.
			daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
			err = upsertInTransaction(tx, daemon.KeaDaemon.KeaDHCPDaemon.ID, daemon.KeaDaemon.KeaDHCPDaemon)
			if err != nil {
				return errors.WithMessagef(err, "problem upserting Kea DHCP daemon %d: %v",
					daemon.KeaDaemon.KeaDHCPDaemon.ID, daemon.KeaDaemon.KeaDHCPDaemon)
			}
		}
	case daemon.Bind9Daemon != nil:
		// Make sure that the bind9_daemon references the daemon.
		daemon.Bind9Daemon.DaemonID = daemon.ID
		err = upsertInTransaction(tx, daemon.Bind9Daemon.ID, daemon.Bind9Daemon)
		if err != nil {
			return errors.WithMessagef(err, "problem upserting BIND 9 daemon %d: %v",
				daemon.Bind9Daemon.ID, daemon.Bind9Daemon)
		}
	case daemon.PDNSDaemon != nil:
		// Make sure that the pdns_daemon references the daemon.
		daemon.PDNSDaemon.DaemonID = daemon.ID
		err = upsertInTransaction(tx, daemon.PDNSDaemon.ID, daemon.PDNSDaemon)
		if err != nil {
			return errors.WithMessagef(err, "problem upserting PowerDNS daemon %d: %v",
				daemon.ID, daemon.PDNSDaemon)
		}
	}

	// Add log targets.
	for _, logTarget := range daemon.LogTargets {
		logTarget.DaemonID = daemon.ID
		err = addLogTarget(tx, logTarget)
		if err != nil {
			return errors.WithMessagef(err, "problem adding log target %v for daemon %d",
				logTarget, daemon.ID)
		}
	}

	return nil
}

// Adds or updates a daemon in the database.
func AddDaemon(dbi dbops.DBI, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addDaemon(tx, daemon)
		})
	}
	return addDaemon(dbi.(*pg.Tx), daemon)
}

// Updates a daemon in a transaction, including dependent Daemon, AccessPoints,
// KeaDaemon, KeaDHCPDaemon and Bind9Daemon if they are not nil.
func updateDaemon(dbi dbops.DBI, daemon *Daemon) error {
	// Update common daemon instance.
	result, err := dbi.Model(daemon).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		return errors.Wrapf(err, "problem updating daemon %d", daemon.ID)
	} else if result.RowsAffected() <= 0 {
		return errors.Wrapf(ErrNotExists, "daemon with ID %d does not exist", daemon.ID)
	}

	// If this is a Kea daemon, we have to update Kea specific tables too.
	switch {
	case daemon.KeaDaemon != nil && daemon.KeaDaemon.ID != 0:
		// Make sure that the KeaDaemon points to the Daemon.
		daemon.KeaDaemon.DaemonID = daemon.ID
		result, err := dbi.Model(daemon.KeaDaemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem updating general Kea-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return errors.Wrapf(ErrNotExists, "Kea daemon with ID %d does not exist", daemon.KeaDaemon.ID)
		}

		// If this is Kea DHCP daemon, there is one more table to update.
		if daemon.KeaDaemon.KeaDHCPDaemon != nil && daemon.KeaDaemon.KeaDHCPDaemon.ID != 0 {
			daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
			result, err := dbi.Model(daemon.KeaDaemon.KeaDHCPDaemon).WherePK().Update()
			if err != nil {
				return errors.Wrapf(err, "problem updating general Kea DHCP information for daemon %d",
					daemon.ID)
			} else if result.RowsAffected() <= 0 {
				return errors.Wrapf(ErrNotExists, "Kea DHCP daemon with ID %d does not exist",
					daemon.KeaDaemon.KeaDHCPDaemon.ID)
			}
		}
	case daemon.Bind9Daemon != nil && daemon.Bind9Daemon.ID != 0:
		// This is Bind9 daemon. Update the Bind9 specific table.
		daemon.Bind9Daemon.DaemonID = daemon.ID
		result, err := dbi.Model(daemon.Bind9Daemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem updating BIND 9-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return errors.Wrapf(ErrNotExists, "BIND 9 daemon with ID %d does not exist", daemon.Bind9Daemon.ID)
		}
	case daemon.PDNSDaemon != nil && daemon.PDNSDaemon.ID != 0:
		// This is PowerDNS daemon. Update the PowerDNS specific table.
		daemon.PDNSDaemon.DaemonID = daemon.ID
		result, err := dbi.Model(daemon.PDNSDaemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem updating PowerDNS-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return errors.Wrapf(ErrNotExists, "PowerDNS daemon with ID %d does not exist", daemon.PDNSDaemon.ID)
		}
	}

	return updateDaemonRelations(dbi, daemon)
}

// Updates the daemon-related entities.
func updateDaemonRelations(dbi dbops.DBI, daemon *Daemon) error {
	// Update the access points.
	for _, accessPoint := range daemon.AccessPoints {
		accessPoint.DaemonID = daemon.ID
		err := addOrUpdateAccessPoint(dbi, accessPoint)
		if err != nil {
			return errors.WithMessagef(err, "problem adding or updating access point %v for daemon %d",
				accessPoint, daemon.ID)
		}
	}
	// Remove access points that are not in the list.
	accessPointTypes := []AccessPointType{}
	for _, accessPoint := range daemon.AccessPoints {
		accessPointTypes = append(accessPointTypes, accessPoint.Type)
	}
	err := deleteAccessPointsExcept(dbi, daemon.ID, accessPointTypes)
	if err != nil {
		return errors.WithMessagef(err, "problem deleting access points for daemon %d", daemon.ID)
	}

	// Update the log targets.
	logTargetIDs := []int64{}
	for _, t := range daemon.LogTargets {
		if t.ID > 0 {
			logTargetIDs = append(logTargetIDs, t.ID)
		}
	}
	err = deleteLogTargetsByDaemonIDExcept(dbi, daemon.ID, logTargetIDs)
	if err != nil {
		return errors.WithMessagef(err, "problem deleting log targets for updated daemon %d",
			daemon.ID)
	}

	// Insert or update log targets.
	for i := range daemon.LogTargets {
		daemon.LogTargets[i].DaemonID = daemon.ID
		err = addOrUpdateLogTarget(dbi, daemon.LogTargets[i])
		if err != nil {
			return errors.WithMessagef(err, "problem altering log target %v for daemon %d",
				daemon.LogTargets[i], daemon.ID)
		}
	}

	return nil
}

// Updates a daemon, including dependent Daemon, KeaDaemon, KeaDHCPDaemon
// and Bind9Daemon if they are not nil.
func UpdateDaemon(dbi dbops.DBI, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateDaemon(tx, daemon)
		})
	}
	return updateDaemon(dbi.(*pg.Tx), daemon)
}

// Updates a daemon statistics information only.
func updateDaemonStatistics(dbi dbops.DBI, daemon *Daemon) error {
	if daemon.Bind9Daemon != nil {
		result, err := dbi.Model(daemon.Bind9Daemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem updating BIND 9-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return errors.Wrapf(ErrNotExists, "BIND 9 daemon with ID %d does not exist", daemon.Bind9Daemon.ID)
		}
	}

	if daemon.KeaDaemon != nil && daemon.KeaDaemon.KeaDHCPDaemon != nil {
		result, err := dbi.Model(daemon.KeaDaemon.KeaDHCPDaemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem updating Kea DHCP-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return errors.Wrapf(ErrNotExists, "Kea DHCP daemon with ID %d does not exist",
				daemon.KeaDaemon.KeaDHCPDaemon.ID)
		}
	}

	return nil
}

// Updates a daemon statistics information only. Wraps the update in a
// transaction if necessary.
func UpdateDaemonStatistics(dbi dbops.DBI, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateDaemonStatistics(tx, daemon)
		})
	}
	return updateDaemonStatistics(dbi.(*pg.Tx), daemon)
}

// Deletes a daemon from the database and its references.
func deleteDaemon(tx *pg.Tx, daemon *Daemon) error {
	result, err := tx.Model(daemon).WherePK().Delete()
	if err != nil {
		return errors.Wrapf(err, "problem deleting daemon: %d", daemon.ID)
	} else if result.RowsAffected() <= 0 {
		return errors.Wrapf(ErrNotExists, "daemon with ID %d does not exist", daemon.ID)
	}
	return nil
}

// Deletes a daemon from the database with all associated access points,
// log targets, KeaDaemon, KeaDHCPDaemon and Bind9Daemon, if they are not nil.
func DeleteDaemon(dbi dbops.DBI, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return deleteDaemon(tx, daemon)
		})
	}
	return deleteDaemon(dbi.(*pg.Tx), daemon)
}

// Deletes the config hash values for all Kea daemons.
func DeleteKeaDaemonConfigHashes(dbi dbops.DBI) error {
	_, err := dbi.Exec("UPDATE kea_daemon SET config_hash = NULL")
	if err != nil {
		return errors.Wrapf(err, "problem deleting Kea config hashes")
	}
	return nil
}

// This is a hook to go-pg that is called just after reading rows from database.
// It reconverts KeaDaemon's configuration from json string maps to the
// expected structure in GO.
func (d *KeaDaemon) AfterScan(ctx context.Context) error {
	if d.Config == nil {
		return nil
	}

	bytes, err := json.Marshal(d.Config)
	if err != nil {
		return errors.Wrapf(err, "problem marshalling Kea config: %+v ", *d.Config)
	}

	err = json.Unmarshal(bytes, d.Config)
	if err != nil {
		return errors.Wrapf(err, "problem unmarshalling Kea config")
	}
	return nil
}

// Returns a slice containing HA information specific for the daemon. This function
// assumes that the daemon has been fetched from the database along with the
// services. It doesn't perform database queries on its own.
func (d *Daemon) GetHAOverview() (overviews []DaemonServiceOverview) {
	for _, service := range d.Services {
		if service.HAService == nil {
			continue
		}
		var overview DaemonServiceOverview
		overview.State = service.GetDaemonHAState(d.ID)
		overview.LastFailureAt = service.GetPartnerHAFailureTime(d.ID)
		overviews = append(overviews, overview)
	}
	return overviews
}

// Sets new configuration of the daemon. This function should be used to set
// new daemon configuration instead of simple configuration assignment because
// it extracts some configuration information and populates to the daemon structures,
// e.g. logging configuration. The config should be a pointer to the keaconfig.Config
// structure. The config_hash is a hash created from the specified configuration.
func (d *Daemon) setKeaConfigWithHash(config *keaconfig.Config, configHash string) error {
	if d.KeaDaemon != nil {
		existingLogTargets := d.LogTargets
		d.LogTargets = []*LogTarget{}
		loggers := config.GetLoggers()
		for _, logger := range loggers {
			targets := NewLogTargetsFromKea(d.ID, logger)
			for i := range targets {
				// For each target check if it already exists and inherit its
				// ID and creation time.
				for _, existingTarget := range existingLogTargets {
					if targets[i].Name == existingTarget.Name &&
						targets[i].Output == existingTarget.Output &&
						existingTarget.DaemonID == d.ID {
						targets[i].ID = existingTarget.ID
						targets[i].DaemonID = d.ID
						targets[i].CreatedAt = existingTarget.CreatedAt
						break
					}
				}
				d.LogTargets = append(d.LogTargets, targets[i])
			}
		}
		d.KeaDaemon.Config = newKeaConfig(config)
		d.KeaDaemon.ConfigHash = configHash
	}
	return nil
}

// Sets new configuration specified as JSON string. The config is set only if
// its hash is different from the existing configuration hash.
func (d *Daemon) SetKeaConfigFromJSON(config []byte) error {
	if d.KeaDaemon == nil {
		// Not a Kea daemon.
		return errors.New("not a Kea daemon")
	}

	hash := keaconfig.NewHasher().Hash(config)
	if d.KeaDaemon.ConfigHash == hash {
		// Configuration is unchanged, nothing to do.
		return nil
	}

	parsedConfig, err := keaconfig.NewConfig(config)
	if err != nil {
		return err
	}

	return d.setKeaConfigWithHash(parsedConfig, hash)
}

// Returns local subnet ID for a given subnet prefix. If subnets have been indexed,
// this function will use the index to find a subnet with the matching prefix. This
// is much faster, but requires that the caller first builds a collection of
// indexed subnets and associates it with appropriate KeaDHCPDaemon instances.
// If the indexes are not built, this function will simply iterate over the
// subnets within the configuration. This is generally much slower and should
// be only used for sporadic calls to GetLocalSubnetID().
// If the matching subnet is not found, the 0 value is returned.
func (d *Daemon) GetLocalSubnetID(prefix string) int64 {
	if d.KeaDaemon == nil {
		return 0
	}
	if d.KeaDaemon.Config != nil {
		if subnet := d.KeaDaemon.Config.GetSubnetByPrefix(prefix); subnet != nil {
			return subnet.GetID()
		}
	}
	return 0
}

// Creates shallow copy of KeaDaemon, i.e. copies Daemon structure and
// nested KeaDaemon structure. The new instance of KeaDaemon is created
// but the pointers under KeaDaemon are inherited from the source.
func ShallowCopyKeaDaemon(daemon *Daemon) *Daemon {
	copied := &Daemon{}
	*copied = *daemon
	if daemon.KeaDaemon != nil {
		copied.KeaDaemon = &KeaDaemon{}
		*copied.KeaDaemon = *daemon.KeaDaemon
	}
	return copied
}

// DaemonTag implementation.

// Returns daemon ID.
func (d Daemon) GetID() int64 {
	return d.ID
}

// Returns daemon name.
func (d Daemon) GetName() daemonname.Name {
	return d.Name
}

// Returns ID of a machine owning the daemon.
func (d Daemon) GetMachineID() int64 {
	return d.MachineID
}
