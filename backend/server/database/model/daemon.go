package dbmodel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// A structure holding Kea DHCP specific information about a daemon. It
// reflects the kea_dhcp_daemon table which extends the daemon and
// kea_daemon tables with the Kea DHCPv4 or DHCPv6 specific information.
type KeaDHCPDaemon struct {
	tableName       struct{} `pg:"kea_dhcp_daemon"` //nolint:unused,structcheck
	ID              int64
	LPS15min        int `pg:"lps15min"`
	LPS24h          int `pg:"lps24h"`
	AddrUtilization int16
	PdUtilization   int16

	KeaDaemonID int64
}

// A structure holding common information for all Kea daemons. It
// reflects the information stored in the kea_daemon table.
type KeaDaemon struct {
	ID     int64
	Config *KeaConfig `pg:",use_zero"`

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

	KeaDaemon   *KeaDaemon
	Bind9Daemon *Bind9Daemon
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
