package dbmodeltest

import (
	"fmt"
	"math/rand"

	"github.com/go-pg/pg/v10"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	dbmodel "isc.org/stork/server/database/model"
)

// Returns 32-bit random integer.
func getRandInt31() int32 {
	return rand.Int31() //nolint:gosec
}

// Returns 64-bit random integer.
func getRandInt63() int64 {
	return rand.Int63() //nolint:gosec
}

// A wrapper for a machine representation in the database.
type Machine struct {
	db *pg.DB
	ID int64
}

// Creates a new machine in the database with a default address and port.
func NewMachine(db *pg.DB) (*Machine, error) {
	m := dbmodel.Machine{
		Address:   fmt.Sprintf("machine%d", getRandInt63()),
		AgentPort: int64(getRandInt31()),
	}
	if err := dbmodel.AddMachine(db, &m); err != nil {
		return nil, err
	}
	machine := &Machine{
		db: db,
		ID: m.ID,
	}
	return machine, nil
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

	daemon := &dbmodel.Daemon{
		MachineID:    m.ID,
		Name:         name,
		Active:       true,
		AccessPoints: ap,
		KeaDaemon: &dbmodel.KeaDaemon{
			KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
		},
	}
	if err := dbmodel.AddDaemon(m.db, daemon); err != nil {
		return nil, err
	}

	return &KeaServer{
		machine:  m,
		DaemonID: daemon.ID,
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
