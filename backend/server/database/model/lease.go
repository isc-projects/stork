package dbmodel

import (
	keadata "isc.org/stork/appdata/kea"
)

// Extends basic Lease information with database specific information.
type Lease struct {
	keadata.Lease
	AppID int64
}
