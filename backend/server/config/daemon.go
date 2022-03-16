package config

import (
	dbmodel "isc.org/stork/server/database/model"
)

// An implementation of the dbmodel.DaemonTag interface used
// by the configuration manager to represent daemons to which
// control commands can be sent.
type Daemon struct {
	ID    int64
	Name  string
	AppID int64
}

// Returns daemon ID.
func (daemon Daemon) GetID() int64 {
	return daemon.ID
}

// Returns daemon name.
func (daemon Daemon) GetName() string {
	return daemon.Name
}

// Returns ID of an app owning the daemon.
func (daemon Daemon) GetAppID() int64 {
	return daemon.AppID
}

// Returns type of an app owning the daemon.
func (daemon Daemon) GetAppType() string {
	switch daemon.Name {
	case "named":
		return dbmodel.AppTypeBind9
	default:
		return dbmodel.AppTypeKea
	}
}
