package config

import (
	pkgerrors "github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Meta field type in the App structure.
type AppMeta struct {
	Version string
}

// Access point structure within the App.
type AccessPoint struct {
	Type              string
	Address           string
	Port              int64
	Key               string
	UseSecureProtocol bool
}

// An implementation of the agentcomm.ControlledApp interface used
// by the configuration manager to represent applications to which
// control commands can be sent. It is a simple structure that can
// be easily marshalled and unmarshalled not only using the json
// decoder but also the mapstructure package.
type App struct {
	ID           int64
	Name         string
	Type         string
	Meta         AppMeta
	Machine      Machine
	AccessPoints []AccessPoint
	Daemons      []Daemon
}

// Returns app ID.
func (app App) GetID() int64 {
	return app.ID
}

// Returns app name.
func (app App) GetName() string {
	return app.Name
}

// Returns app type.
func (app App) GetType() string {
	return app.Type
}

// Returns app version.
func (app App) GetVersion() string {
	return app.Meta.Version
}

// Returns app control access point including control address, port, key and
// the flag indicating if the connection is secure.
func (app App) GetControlAccessPoint() (address string, port int64, key string, secure bool, err error) {
	for _, ap := range app.AccessPoints {
		if ap.Type == dbmodel.AccessPointControl {
			address = ap.Address
			port = ap.Port
			key = ap.Key
			secure = ap.UseSecureProtocol
			return
		}
	}
	err = pkgerrors.Errorf("no access point of type %s found for app id %d", dbmodel.AccessPointControl, app.ID)
	return
}

// Returns MachineTag interface to the machine owning the app.
func (app App) GetMachineTag() dbmodel.MachineTag {
	return app.Machine
}

// Returns DaemonTag interfaces to the daemons owned by the app.
func (app App) GetDaemonTags() (tags []dbmodel.DaemonTag) {
	for i := range app.Daemons {
		tags = append(tags, app.Daemons[i])
	}
	return
}
