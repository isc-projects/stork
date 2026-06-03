package dbmodeltest

import (
	"fmt"
	"math/rand"

	"github.com/go-pg/pg/v10"
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

// A wrapper for a daemon.
type Daemon struct {
	machine  *Machine
	DaemonID int64
}

// Creates a new machine in the database with a default address and port.
func NewMachine(db *pg.DB) (*Machine, error) {
	m := dbmodel.Machine{
		Address:    fmt.Sprintf("machine%d", getRandInt63()),
		AgentPort:  int64(getRandInt31()),
		Authorized: true,
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

// Returns a machine representation from the database.
func (m *Machine) GetMachine() (*dbmodel.Machine, error) {
	return dbmodel.GetMachineByID(m.db, m.ID)
}

// Returns a machine the server belongs to.
func (server *Daemon) GetMachine() (*dbmodel.Machine, error) {
	return server.machine.GetMachine()
}

// Returns a daemon the server belongs to.
func (server *Daemon) GetDaemon() (*dbmodel.Daemon, error) {
	daemon, err := dbmodel.GetKeaDaemonByID(server.machine.db, server.DaemonID)
	if err != nil {
		return nil, err
	}
	return daemon, nil
}
