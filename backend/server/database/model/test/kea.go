package dbmodeltest

import (
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// A wrapper for a Kea app representation in the database.
type Kea struct {
	machine *Machine
	ID      int64
}

// Creates a machine and a Kea app instance running on it in the database.
func NewKea(db *pg.DB) (*Kea, error) {
	machine, err := NewMachine(db)
	if err != nil {
		return nil, err
	}
	kea, err := machine.NewKea()
	return kea, err
}

// Returns a machine instance from the database that the Kea app belongs to.
func (kea *Kea) GetMachine() (*dbmodel.Machine, error) {
	return dbmodel.GetMachineByID(kea.machine.db, kea.machine.ID)
}

// Creates a new Kea server instance for the Kea app.
func (kea *Kea) newServer(name string) (*KeaServer, error) {
	a, err := dbmodel.GetAppByID(kea.machine.db, kea.ID)
	if err != nil {
		return nil, err
	}
	d := dbmodel.NewKeaDaemon(name, true)
	a.Daemons = append(a.Daemons, d)
	addedDaemons, _, err := dbmodel.UpdateApp(kea.machine.db, a)
	if err != nil {
		return nil, err
	}
	server := &KeaServer{
		kea: kea,
		ID:  addedDaemons[0].ID,
	}
	return server, nil
}

// Creates DHCOPv4 server instance for the Kea app.
func (kea *Kea) NewKeaDHCPv4Server() (*KeaServer, error) {
	return kea.newServer(dbmodel.DaemonNameDHCPv4)
}

// Creates DHCPv6 server instance for the Kea app.
func (kea *Kea) NewKeaDHCPv6Server() (*KeaServer, error) {
	return kea.newServer(dbmodel.DaemonNameDHCPv6)
}
