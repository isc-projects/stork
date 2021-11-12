package dumps

import (
	"fmt"

	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The dump of the machine database entry.
type MachineDump struct {
	BasicDump
	machine *dbmodel.Machine
}

func NewMachineDump(m *dbmodel.Machine) *MachineDump {
	return &MachineDump{
		*NewBasicDump("machine"),
		m,
	}
}

func (d *MachineDump) Execute() error {
	for _, app := range d.machine.Apps {
		for _, daemon := range app.Daemons {
			if daemon.KeaDaemon != nil && daemon.KeaDaemon.Config != nil {
				storkutil.HideSensitiveData((*map[string]interface{})(daemon.KeaDaemon.Config))
			}
		}
	}

	d.AppendArtifact(NewBasicStructArtifact(
		fmt.Sprintf("%d-%s", d.machine.ID, d.machine.Address),
		d.machine,
	))

	return nil
}
