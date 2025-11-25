package daemons

import (
	"isc.org/stork/server/daemons/bind9"
	"isc.org/stork/server/daemons/kea"
)

// Collection of pullers used by the server.
type Pullers struct {
	StatePuller      *StatePuller
	Bind9StatsPuller *bind9.StatsPuller
	KeaStatsPuller   *kea.StatsPuller
	KeaHostsPuller   *kea.HostsPuller
	HAStatusPuller   *kea.HAStatusPuller
}
