package dumper

import (
	"github.com/go-pg/pg/v9"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dump"
)

// Initialize (construct) the dump instances.
type factory struct {
	db              *pg.DB
	m               *dbmodel.Machine
	connectedAgents agentcomm.ConnectedAgents
}

func newFactory(db *pg.DB, m *dbmodel.Machine, agents agentcomm.ConnectedAgents) factory {
	return factory{
		db:              db,
		m:               m,
		connectedAgents: agents,
	}
}

// Construct all supported dumps.
func (f *factory) all() []dump.Dump {
	return []dump.Dump{
		dump.NewMachineDump(f.m),
		dump.NewEventsDump(f.db, f.m),
		dump.NewLogsDump(f.m, f.connectedAgents),
		dump.NewSettingsDump(f.db),
	}
}
