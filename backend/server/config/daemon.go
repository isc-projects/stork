package config

import (
	dbmodel "isc.org/stork/server/database/model"
)

// It's used by the configuration manager to represent daemons to which
// control commands can be sent.
type Daemon struct {
	ID    int64
	Name  string
	AppID int64
}

// An implementation of the dbmodel.DaemonTag interface.
type daemonTag struct {
	ID        int64
	Name      string
	AppID     int64
	MachineID int64
}

// Construct a new daemon tag instance from the daemon object.
func newDaemonTag(daemon Daemon, machineID int64) daemonTag {
	return daemonTag{
		ID:        daemon.ID,
		Name:      daemon.Name,
		AppID:     daemon.AppID,
		MachineID: machineID,
	}
}

// Returns daemon ID.
func (t daemonTag) GetID() int64 {
	return t.ID
}

// Returns daemon name.
func (t daemonTag) GetName() string {
	return t.Name
}

// Returns ID of an app owning the daemon.
func (t daemonTag) GetAppID() int64 {
	return t.AppID
}

// Returns type of an app owning the daemon. It returns "unknown"
// when daemon has unrecognized name.
func (t daemonTag) GetAppType() string {
	switch t.Name {
	case dbmodel.DaemonNameBind9:
		return dbmodel.AppTypeBind9
	case dbmodel.DaemonNameDHCPv4, dbmodel.DaemonNameDHCPv6, dbmodel.DaemonNameD2, dbmodel.DaemonNameCA:
		return dbmodel.AppTypeKea
	default:
		return "unknown"
	}
}

// Returns ID of a machine owning the daemon.
func (t daemonTag) GetMachineID() *int64 {
	return &t.MachineID
}
