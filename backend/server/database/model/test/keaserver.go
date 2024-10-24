package dbmodeltest

import (
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// A wrapper for a Kea daemon.
type KeaServer struct {
	kea *Kea
	ID  int64
}

// Creates new Kea app and a DHCPv4 server daemon in the database.
func NewKeaDHCPv4Server(db *pg.DB) (*KeaServer, error) {
	kea, err := NewKea(db)
	if err != nil {
		return nil, err
	}
	dhcp4, err := kea.NewKeaDHCPv4Server()
	if err != nil {
		return nil, err
	}
	return dhcp4, nil
}

// Creates new Kea app and a DHCPv6 server daemon in the database.
func NewKeaDHCPv6Server(db *pg.DB) (*KeaServer, error) {
	kea, err := NewKea(db)
	if err != nil {
		return nil, err
	}
	dhcp6, err := kea.NewKeaDHCPv6Server()
	if err != nil {
		return nil, err
	}
	return dhcp6, nil
}

// Applies a new configuration in the Kea server.
func (server *KeaServer) Configure(config string) error {
	d, err := dbmodel.GetDaemonByID(server.kea.machine.db, server.ID)
	if err != nil {
		return err
	}
	err = d.SetConfigFromJSON(config)
	if err != nil {
		return err
	}
	return dbmodel.UpdateDaemon(server.kea.machine.db, d)
}

// Sets an arbitrary Kea server version.
func (server *KeaServer) SetVersion(version string) error {
	d, err := dbmodel.GetDaemonByID(server.kea.machine.db, server.ID)
	if err != nil {
		return err
	}
	d.Version = version
	return dbmodel.UpdateDaemon(server.kea.machine.db, d)
}

// Returns a machine the Kea server belongs to.
func (server *KeaServer) GetMachine() (*dbmodel.Machine, error) {
	return server.kea.GetMachine()
}

// Returns a Kea app the Kea server belongs to.
func (server *KeaServer) GetKea() (*dbmodel.App, error) {
	return dbmodel.GetAppByID(server.kea.machine.db, server.kea.ID)
}
