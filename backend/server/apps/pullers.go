package apps

import (
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
)

// Collection of pullers used by the server.
type Pullers struct {
	AppsStatePuller  *StatePuller
	Bind9StatsPuller *bind9.StatsPuller
	KeaStatsPuller   *kea.StatsPuller
	KeaHostsPuller   *kea.HostsPuller
	HAStatusPuller   *kea.HAStatusPuller
}
