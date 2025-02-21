package appstest

import (
	"github.com/go-pg/pg/v10"
	keaconfig "isc.org/stork/appcfg/kea"
	agentcomm "isc.org/stork/server/agentcomm"
	"isc.org/stork/server/config"
)

// Implements ManagerAccessors interface for unit tests.
type ManagerAccessorsWrapper struct {
	DB           *pg.DB
	Agents       agentcomm.ConnectedAgents
	DefLookup    keaconfig.DHCPOptionDefinitionLookup
	DaemonLocker config.DaemonLocker
}

// Returns an instance of the database handler used by the configuration manager.
func (w ManagerAccessorsWrapper) GetDB() *pg.DB {
	return w.DB
}

// Returns an interface to the agents the manager communicates with.
func (w ManagerAccessorsWrapper) GetConnectedAgents() agentcomm.ConnectedAgents {
	return w.Agents
}

// Returns an interface to the instance providing the DHCP option definition
// lookup logic.
func (w ManagerAccessorsWrapper) GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return w.DefLookup
}

// Returns an interface to the instance providing the daemon configurations'
// locking mechanism.
func (w ManagerAccessorsWrapper) GetDaemonLocker() config.DaemonLocker {
	return w.DaemonLocker
}
