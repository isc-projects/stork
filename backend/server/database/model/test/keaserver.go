package dbmodeltest

import (
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// A wrapper for a Kea daemon.
type KeaServer struct {
	machine  *Machine
	DaemonID int64
}

// Creates new Kea daemon and a DHCPv4 server daemon in the database.
func NewKeaDHCPv4Server(db *pg.DB) (*KeaServer, error) {
	m, err := NewMachine(db)
	if err != nil {
		return nil, err
	}
	dhcp4, err := m.NewKeaDHCPv4Server()
	if err != nil {
		return nil, err
	}
	return dhcp4, nil
}

// Creates new Kea daemon and a DHCPv6 server daemon in the database.
func NewKeaDHCPv6Server(db *pg.DB) (*KeaServer, error) {
	m, err := NewMachine(db)
	if err != nil {
		return nil, err
	}
	dhcp6, err := m.NewKeaDHCPv6Server()
	if err != nil {
		return nil, err
	}
	return dhcp6, nil
}

// Applies a new configuration in the Kea server.
func (server *KeaServer) Configure(config string) error {
	d, err := dbmodel.GetKeaDaemonByID(server.machine.db, server.DaemonID)
	if err != nil {
		return err
	}
	err = d.SetKeaConfigFromJSON([]byte(config))
	if err != nil {
		return err
	}
	return dbmodel.UpdateDaemon(server.machine.db, d)
}

// Sets an arbitrary Kea server version.
func (server *KeaServer) SetVersion(version string) error {
	d, err := dbmodel.GetKeaDaemonByID(server.machine.db, server.DaemonID)
	if err != nil {
		return err
	}
	d.Version = version
	return dbmodel.UpdateDaemon(server.machine.db, d)
}

// Returns a machine the Kea server belongs to.
func (server *KeaServer) GetMachine() (*dbmodel.Machine, error) {
	machine, err := dbmodel.GetMachineByID(server.machine.db, server.machine.ID)
	if err != nil {
		return nil, err
	}
	return machine, nil
}

// Returns a daemon the Kea server belongs to.
func (server *KeaServer) GetDaemon() (*dbmodel.Daemon, error) {
	daemon, err := dbmodel.GetKeaDaemonByID(server.machine.db, server.DaemonID)
	if err != nil {
		return nil, err
	}
	return daemon, nil
}
