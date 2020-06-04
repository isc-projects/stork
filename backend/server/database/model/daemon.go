package dbmodel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
)

const (
	DaemonNameBind9  = "named"
	DaemonNameDHCPv4 = "dhcp4"
	DaemonNameDHCPv6 = "dhcp6"
)

// A structure reflecting Kea DHCP stats for daemon. It is stored
// as a JSONB value in SQL and unmarshalled in this structure.
type KeaDHCPDaemonStats struct {
	LPS15min        int `pg:"lps15min"`
	LPS24h          int `pg:"lps24h"`
	AddrUtilization int16
	PdUtilization   int16
}

// A structure holding Kea DHCP specific information about a daemon. It
// reflects the kea_dhcp_daemon table which extends the daemon and
// kea_daemon tables with the Kea DHCPv4 or DHCPv6 specific information.
type KeaDHCPDaemon struct {
	tableName   struct{} `pg:"kea_dhcp_daemon"` //nolint:unused,structcheck
	ID          int64
	KeaDaemonID int64
	Stats       KeaDHCPDaemonStats
}

// A structure holding common information for all Kea daemons. It
// reflects the information stored in the kea_daemon table.
type KeaDaemon struct {
	ID       int64
	Config   *KeaConfig `pg:",use_zero"`
	DaemonID int64

	KeaDHCPDaemon *KeaDHCPDaemon
}

// A structure reflecting BIND9 stats for a daemon. It is stored
// as a JSONB value in SQL and unarshalled to this structure.
type Bind9DaemonStats struct {
	ZoneCount          int64
	AutomaticZoneCount int64
	CacheHits          int64
	CacheMisses        int64
	CacheHitRatio      float64
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
	Version         string
	ExtendedVersion string
	Uptime          int64
	CreatedAt       time.Time
	ReloadedAt      time.Time

	AppID int64
	App   *App

	Services []*Service `pg:"many2many:daemon_to_service,fk:daemon_id,joinFK:service_id"`

	KeaDaemon   *KeaDaemon
	Bind9Daemon *Bind9Daemon
}

// Structure representing HA service information displayed for the daemon
// in the dashboard.
type DaemonServiceOverview struct {
	State         string
	LastFailureAt time.Time
}

// Creates an instance of a Kea daemon. If the daemon name is dhcp4 or
// dhcp6, the instance of the KeaDHCPDaemon is also created.
func NewKeaDaemon(name string, active bool) *Daemon {
	daemon := &Daemon{
		Name:      name,
		KeaDaemon: &KeaDaemon{},
		Active:    active,
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
		Bind9Daemon: &Bind9Daemon{},
	}
	return daemon
}

// Updates a daemon, including dependent Daemon, KeaDaemon, KeaDHCPDaemon
// and Bind9Daemon if they are not nil.
func UpdateDaemon(dbIface interface{}, daemon *Daemon) error {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	// Always rollback when this function ends. If the changes get committed
	// first this is no-op.
	defer rollback()

	// Update common daemon instance.
	_, err = tx.Model(daemon).WherePK().Update()
	if err != nil {
		return errors.Wrapf(err, "problem with updating daemon %d", daemon.ID)
	}

	// If this is a Kea daemon, we have to update Kea specific tables too.
	if daemon.KeaDaemon != nil && daemon.KeaDaemon.ID != 0 {
		// Make sure that the KeaDaemon points to the Daemon.
		daemon.KeaDaemon.DaemonID = daemon.ID
		_, err = tx.Model(daemon.KeaDaemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem with updating general Kea specific information for daemon %d",
				daemon.ID)
		}

		// If this is Kea DHCP daemon, there is one more table to update.
		if daemon.KeaDaemon.KeaDHCPDaemon != nil && daemon.KeaDaemon.KeaDHCPDaemon.ID != 0 {
			daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
			_, err = tx.Model(daemon.KeaDaemon.KeaDHCPDaemon).WherePK().Update()
			if err != nil {
				return errors.Wrapf(err, "problem with updating general Kea DHCP information for daemon %d",
					daemon.ID)
			}
		}
	} else if daemon.Bind9Daemon != nil && daemon.Bind9Daemon.ID != 0 {
		// This is Bind9 daemon. Update the Bind9 specific table.
		daemon.Bind9Daemon.DaemonID = daemon.ID
		_, err = tx.Model(daemon.Bind9Daemon).WherePK().Update()
		if err != nil {
			return errors.Wrapf(err, "problem with updating Bind9 specific information for daemon %d",
				daemon.ID)
		}
	}

	err = commit()
	if err != nil {
		err = errors.WithMessagef(err, "problem with committing daemon %d after update", daemon.ID)
	}

	return err
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
		return errors.Wrapf(err, "problem with marshalling Kea config: %+v ", *d.Config)
	}

	err = json.Unmarshal(bytes, d.Config)
	if err != nil {
		return errors.Wrapf(err, "problem with unmarshalling Kea config")
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
