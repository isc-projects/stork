package dbmodel

import (
	"time"
)

// A structure reflecting base_daemon SQL table. This table holds
// generic information about the daemon such as ID, daemon name.
type BaseDaemon struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	AppID     int64
	App       *App
	ServiceID int64
	Service   *BaseService
}

// A structure holding Kea DHCP specific information about the daemon. It
// reflects the kea_dhcp_daemon table which extends the base_daemon table
// with Kea DHCPv4 or DHCPv6 specific information. It is embedded in the
// Daemon structure.
type KeaDhcpDaemon struct {
	ID          int64
	DaemonID    int64
	Daemon      *BaseDaemon
	HAServiceID int64
	HAService   *BaseHAService
	// TODO
	// Active...
	// StatusCollectedAt time.Time
	// State             string
	LPS15min    int
	LPS24h      int
	Utilization int16
}

// A structure reflecting all SQL tables holding information about the
// daemons of various types. It embeds the BaseDaemon structure which
// holds the basic information about the daemon. It also embeds
// KeaDhcpDaemon structure which holds Kea DHCP specific information
// if the service is of the Kea DHCPv4 or DHCPv6 type. It is nil
// if the daemon is not of the Kea DHCP type. This structure is
// to be extended with additional structures as more daemon types
// are defined.
type Daemon struct {
	BaseDaemon
	KeaDhcpDaemon *KeaDhcpDaemon
}
