package dump

import (
	"fmt"

	dbmodel "isc.org/stork/server/database/model"
)

// The dump of the machine database entry.
type MachineDump struct {
	BasicDump
	machine *dbmodel.Machine
}

// Constructs the machine dump.
func NewMachineDump(m *dbmodel.Machine) *MachineDump {
	return &MachineDump{
		*NewBasicDump("machine"),
		m,
	}
}

// Dumps the machine instance provided in the constructor.
// It removes the sensitive data from the dumped data.
// The removed data:
//
// - Agent token
// - The values for restricted keys from Kea daemon configurations.
func (d *MachineDump) Execute() error {
	// Hide agent tokens
	d.machine.AgentToken = ""
	// Hide sensitive data from the daemon configurations
	for _, app := range d.machine.Apps {
		for _, daemon := range app.Daemons {
			if daemon.KeaDaemon != nil && daemon.KeaDaemon.Config != nil {
				daemon.KeaDaemon.Config.HideSensitiveData()
			}
		}
	}

	d.AppendArtifact(NewBasicStructArtifact(
		fmt.Sprintf("%d-%s", d.machine.ID, d.machine.Address),
		d.machine,
	))

	return nil
}
