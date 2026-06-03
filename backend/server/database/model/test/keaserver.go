package dbmodeltest

import (
	"github.com/go-pg/pg/v10"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	dbmodel "isc.org/stork/server/database/model"
)

// A wrapper for a Kea daemon.
type KeaServer struct {
	Daemon
}

// Creates new Kea daemon instance in the machine.
func (m *Machine) newKeaDaemon(name daemonname.Name) (*KeaServer, error) {
	ap := []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "localhost",
		Port:     int64(getRandInt31()),
		Key:      "",
		Protocol: protocoltype.HTTPS,
	}}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{ID: m.ID}, name, true, ap)
	if err := dbmodel.AddDaemon(m.db, daemon); err != nil {
		return nil, err
	}

	return &KeaServer{
		Daemon: Daemon{
			machine:  m,
			DaemonID: daemon.ID,
		},
	}, nil
}

// Creates DHCPPv4 server instance for the Kea daemon.
func (m *Machine) NewKeaDHCPv4Server() (*KeaServer, error) {
	return m.newKeaDaemon(daemonname.DHCPv4)
}

// Creates DHCPv6 server instance for the Kea daemon.
func (m *Machine) NewKeaDHCPv6Server() (*KeaServer, error) {
	return m.newKeaDaemon(daemonname.DHCPv6)
}

// Creates CA instance for the Kea daemon.
func (m *Machine) NewKeaCA() (*KeaServer, error) {
	return m.newKeaDaemon(daemonname.CA)
}

// Creates CA server instance for the Kea daemon.
func (m *Machine) NewKeaCAServer() (*KeaServer, error) {
	return m.newKeaDaemon(daemonname.CA)
}

// Creates D2 server instance for the Kea daemon.
func (m *Machine) NewKeaD2Server() (*KeaServer, error) {
	return m.newKeaDaemon(daemonname.D2)
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

// Creates a new Kea daemon and a CA server daemon in the database.
func NewKeaCAServer(db *pg.DB) (*KeaServer, error) {
	m, err := NewMachine(db)
	if err != nil {
		return nil, err
	}
	daemon, err := m.NewKeaCAServer()
	if err != nil {
		return nil, err
	}
	return daemon, nil
}

// Creates a new Kea daemon and a D2 server daemon in the database.
func NewKeaD2Server(db *pg.DB) (*KeaServer, error) {
	m, err := NewMachine(db)
	if err != nil {
		return nil, err
	}
	daemon, err := m.NewKeaD2Server()
	if err != nil {
		return nil, err
	}
	return daemon, nil
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
