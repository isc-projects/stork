package dump

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// Dumps the services associated with the machine.
type ServicesDump struct {
	BasicDump
	db        *pg.DB
	machineID int64
}

// Constructs new services dump instance.
func NewServicesDump(db *pg.DB, machine *dbmodel.Machine) *ServicesDump {
	return &ServicesDump{
		*NewBasicDump("services"),
		db, machine.ID,
	}
}

// Executes the dump of the services.
func (d *ServicesDump) Execute() error {
	services, err := dbmodel.GetDetailedServicesByMachineID(d.db, d.machineID)
	if err != nil {
		return err
	}

	for i := range services {
		name := fmt.Sprintf("s-%d-%s", services[i].ID, services[i].Name)
		d.AppendArtifact(NewBasicStructArtifact(
			name, &services[i],
		))
	}
	return nil
}
