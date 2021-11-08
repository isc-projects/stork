package dumps

import (
	"fmt"

	dbmodel "isc.org/stork/server/database/model"
)

// The dump of the machine database entry.
type MachineDump struct {
	BasicDump
}

func NewMachineDump(m *dbmodel.Machine) *MachineDump {
	return &MachineDump{
		*NewBasicDump("machine",
			NewBasicStructArtifact(
				fmt.Sprintf("%d-%s", m.ID, m.Address),
				m,
			),
		),
	}
}
