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

// Creates new Kea app instance in the machine.
func (machine *Machine) NewKea() (*Kea, error) {
	ap := []*dbmodel.AccessPoint{}
	ap = dbmodel.AppendAccessPoint(ap, dbmodel.AccessPointControl, "localhost", "", int64(getRandInt31()), true)

	keaApp := dbmodel.App{
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		Name:         fmt.Sprintf("dhcp%d", getRandInt31()),
		Active:       true,
		AccessPoints: ap,
		Daemons:      []*dbmodel.Daemon{},
	}
	if _, err := dbmodel.AddApp(machine.db, &keaApp); err != nil {
		return nil, err
	}
	kea := &Kea{
		machine: machine,
		ID:      keaApp.ID,
	}
	return kea, nil
}
