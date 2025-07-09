package bind9config

// Checks if the zone is a RPZ.
func (rp *ResponsePolicy) IsRPZ(zoneName string) bool {
	for _, zone := range rp.Zones {
		if zone.Zone == zoneName {
			return true
		}
	}
	return false
}
