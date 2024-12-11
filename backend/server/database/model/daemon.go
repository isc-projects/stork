package dbmodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/appdata/bind9stats"
	dbops "isc.org/stork/server/database"
)

// Valid daemon names.
const (
	DaemonNameBind9  = "named"
	DaemonNameDHCPv4 = "dhcp4"
	DaemonNameDHCPv6 = "dhcp6"
	DaemonNameD2     = "d2"
	DaemonNameCA     = "ca"
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
	NamedStats         *bind9stats.Bind9NamedStats
}

// A structure holding BIND9 daemon specific information.
type Bind9Daemon struct {
	ID       int64
	DaemonID int64
	Stats    Bind9DaemonStats
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
	Name            string
	Active          bool `pg:",use_zero"`
	Monitored       bool `pg:",use_zero"`
	Version         string
	ExtendedVersion string
	Uptime          int64
	CreatedAt       time.Time
	ReloadedAt      time.Time

	AppID int64
	App   *App `pg:"rel:has-one"`

	Services []*Service `pg:"many2many:daemon_to_service,fk:daemon_id,join_fk:service_id"`

	LogTargets []*LogTarget `pg:"rel:has-many"`

	KeaDaemon   *KeaDaemon   `pg:"rel:belongs-to"`
	Bind9Daemon *Bind9Daemon `pg:"rel:belongs-to"`

	ConfigReview *ConfigReview `pg:"rel:belongs-to"`
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
	GetName() string
	GetAppID() int64
	GetAppType() AppType
	GetMachineID() *int64
}

// Creates an instance of a Kea daemon. If the daemon name is dhcp4 or
// dhcp6, the instance of the KeaDHCPDaemon is also created.
func NewKeaDaemon(name string, active bool) *Daemon {
	daemon := &Daemon{
		Name:      name,
		Active:    active,
		Monitored: true,
		KeaDaemon: &KeaDaemon{},
	}
	if name == DaemonNameDHCPv4 || name == DaemonNameDHCPv6 {
		daemon.KeaDaemon.KeaDHCPDaemon = &KeaDHCPDaemon{}
	}
	return daemon
}

// Creates an instance of the Bind9 daemon.
func NewBind9Daemon(active bool) *Daemon {
	daemon := &Daemon{
		Name:        DaemonNameBind9,
		Active:      active,
		Monitored:   true,
		Bind9Daemon: &Bind9Daemon{},
	}
	return daemon
}

// Get daemon by ID.
func GetDaemonByID(dbi pg.DBI, id int64) (*Daemon, error) {
	daemon := Daemon{}
	q := dbi.Model(&daemon)
	q = q.Relation("App.AccessPoints")
	q = q.Relation("App.Machine")
	q = q.Relation("KeaDaemon")
	q = q.Where("daemon.id = ?", id)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting daemon %v", id)
	}
	return &daemon, nil
}

// Get selected daemons by their ids.
func GetDaemonsByIDs(dbi pg.DBI, ids []int64) (daemons []Daemon, err error) {
	err = dbi.Model(&daemons).
		Relation("App.AccessPoints").
		Relation("App.Machine").
		Relation("KeaDaemon.KeaDHCPDaemon").
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
		return nil, pkgerrors.Wrapf(err, "problem selecting daemons with IDs: %s",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Get all Kea DHCP daemons.
func GetKeaDHCPDaemons(dbi pg.DBI) (daemons []Daemon, err error) {
	err = dbi.Model(&daemons).
		Relation("App").
		Relation("KeaDaemon.KeaDHCPDaemon").
		Where("daemon.name ILIKE 'dhcp%'").
		OrderExpr("daemon.id ASC").
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		err = nil
	} else {
		err = pkgerrors.Wrapf(err, "problem with getting Kea DHCP daemons")
	}
	return
}

// Select one or more daemons for update. The main use case for this function is
// to prevent modifications and deletions of the daemons while the server inserts
// config reports for them. It must be called within a transaction and the selected
// rows remain locked until the transaction is committed or rolled back. Due to the
// limitations of the go-pg library, this function does not select all columns in the
// joined tables (machine and app).
func GetDaemonsForUpdate(tx *pg.Tx, daemonsToSelect []*Daemon) ([]*Daemon, error) {
	var daemons []*Daemon

	// It is an error when no daemons are specified. Typically it will be one
	// daemon but there can be more.
	if len(daemonsToSelect) == 0 {
		return daemons, pkgerrors.New("no daemons specified for selection for update")
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
		ColumnExpr("app.id AS app__id").
		ColumnExpr("machine.id AS app__machine__id").
		Join("INNER JOIN app ON app.id = daemon.app_id").
		Join("INNER JOIN machine ON app.machine_id = machine.id").
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
		return nil, pkgerrors.Wrapf(err, "problem selecting daemons with IDs: %s, for update",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Select one or more Kea daemons for update. The main use case for this function is
// to prevent modifications and deletions of the Kea daemons while the server inserts
// config reports for them. It must be called within a transaction and the selected
// rows remain locked until the transaction is committed or rolled back. Due to the
// limitations of the go-pg library, this function does not select all columns in the
// joined tables (machine, app and kea_daemon). It selects the config and the
// config_hash columns from the kea_daemon so the configreview implementation can
// verify that the configurations were not changed during the config review process.
// For other joined tables, this function merely returns id columns values.
func GetKeaDaemonsForUpdate(tx *pg.Tx, daemonsToSelect []*Daemon) ([]*Daemon, error) {
	var daemons []*Daemon

	// It is an error when no daemons are specified. Typically it will be one
	// daemon but there can be more.
	if len(daemonsToSelect) == 0 {
		return daemons, pkgerrors.New("no Kea daemons specified for selection for update")
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
		ColumnExpr("app.id AS app__id").
		ColumnExpr("machine.id AS app__machine__id").
		Join("INNER JOIN kea_daemon ON kea_daemon.daemon_id = daemon.id").
		Join("INNER JOIN app ON app.id = daemon.app_id").
		Join("INNER JOIN machine ON app.machine_id = machine.id").
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
		return nil, pkgerrors.Wrapf(err, "problem selecting Kea daemons with IDs: %s, for update",
			strings.Join(sids, ", "))
	}
	return daemons, nil
}

// Updates a daemon in a transaction, including dependent Daemon,
// KeaDaemon, KeaDHCPDaemon and Bind9Daemon if they are not nil.
func updateDaemon(tx *pg.Tx, daemon *Daemon) error {
	// Update common daemon instance.
	result, err := tx.Model(daemon).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem updating daemon %d", daemon.ID)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "daemon with ID %d does not exist", daemon.ID)
	}

	// If this is a Kea daemon, we have to update Kea specific tables too.
	if daemon.KeaDaemon != nil && daemon.KeaDaemon.ID != 0 {
		// Make sure that the KeaDaemon points to the Daemon.
		daemon.KeaDaemon.DaemonID = daemon.ID
		result, err := tx.Model(daemon.KeaDaemon).WherePK().Update()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem updating general Kea-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return pkgerrors.Wrapf(ErrNotExists, "Kea daemon with ID %d does not exist", daemon.KeaDaemon.ID)
		}

		// If this is Kea DHCP daemon, there is one more table to update.
		if daemon.KeaDaemon.KeaDHCPDaemon != nil && daemon.KeaDaemon.KeaDHCPDaemon.ID != 0 {
			daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
			result, err := tx.Model(daemon.KeaDaemon.KeaDHCPDaemon).WherePK().Update()
			if err != nil {
				return pkgerrors.Wrapf(err, "problem updating general Kea DHCP information for daemon %d",
					daemon.ID)
			} else if result.RowsAffected() <= 0 {
				return pkgerrors.Wrapf(ErrNotExists, "Kea DHCP daemon with ID %d does not exist",
					daemon.KeaDaemon.KeaDHCPDaemon.ID)
			}
		}
	} else if daemon.Bind9Daemon != nil && daemon.Bind9Daemon.ID != 0 {
		// This is Bind9 daemon. Update the Bind9 specific table.
		daemon.Bind9Daemon.DaemonID = daemon.ID
		result, err := tx.Model(daemon.Bind9Daemon).WherePK().Update()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem updating BIND 9-specific information for daemon %d",
				daemon.ID)
		} else if result.RowsAffected() <= 0 {
			return pkgerrors.Wrapf(ErrNotExists, "BIND 9 daemon with ID %d does not exist", daemon.Bind9Daemon.ID)
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

// Deletes the config hash values for all Kea daemons.
func DeleteKeaDaemonConfigHashes(dbi dbops.DBI) error {
	_, err := dbi.Exec("UPDATE kea_daemon SET config_hash = NULL")
	if err != nil {
		return pkgerrors.Wrapf(err, "problem deleting Kea config hashes")
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
		return pkgerrors.Wrapf(err, "problem marshalling Kea config: %+v ", *d.Config)
	}

	err = json.Unmarshal(bytes, d.Config)
	if err != nil {
		return pkgerrors.Wrapf(err, "problem unmarshalling Kea config")
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
// e.g. logging configuration. The config should be a pointer to the KeaConfig
// structure. The config_hash is a hash created from the specified configuration.
func (d *Daemon) SetConfigWithHash(config *KeaConfig, configHash string) error {
	if d.KeaDaemon != nil {
		existingLogTargets := d.LogTargets
		d.LogTargets = []*LogTarget{}
		loggers := config.GetLoggers()
		for _, logger := range loggers {
			targets := NewLogTargetsFromKea(logger)
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
					}
				}
				d.LogTargets = append(d.LogTargets, targets[i])
			}
		}
		d.KeaDaemon.Config = config
		d.KeaDaemon.ConfigHash = configHash
	}
	return nil
}

// Sets new configuration of the daemon with empty hash.
func (d *Daemon) SetConfig(config *KeaConfig) error {
	return d.SetConfigWithHash(config, "")
}

// Sets new configuration specified as JSON string. Internally, it calls
// SetConfig after parsing the JSON configuration.
func (d *Daemon) SetConfigFromJSON(config string) error {
	if d.KeaDaemon != nil {
		parsedConfig, err := NewKeaConfigFromJSON(config)
		if err != nil {
			return err
		}

		return d.SetConfigWithHash(parsedConfig, keaconfig.NewHasher().Hash(config))
	}
	return nil
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
func (d Daemon) GetName() string {
	return d.Name
}

// Returns ID of an app owning the daemon.
func (d Daemon) GetAppID() int64 {
	return d.AppID
}

// Returns type of an app owning the daemon.
func (d Daemon) GetAppType() (apptype AppType) {
	switch {
	case d.App != nil:
		apptype = d.App.Type
	case d.KeaDaemon != nil:
		apptype = AppTypeKea
	case d.Bind9Daemon != nil:
		apptype = AppTypeBind9
	}
	return
}

// Returns ID of a machine owning the daemon or nil if the app is unknown.
func (d Daemon) GetMachineID() *int64 {
	if d.App == nil {
		return nil
	}
	return &d.App.MachineID
}
