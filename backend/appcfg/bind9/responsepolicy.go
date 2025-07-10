package bind9config

import "strings"

// Checks if the zone is a RPZ by running a case-insensitive comparison
// of the zone name with the zone names in the response-policy clause.
func (rp *ResponsePolicy) IsRPZ(zoneName string) bool {
	for _, zone := range rp.Zones {
		if strings.EqualFold(zone.Zone, zoneName) {
			return true
		}
	}
	return false
}
