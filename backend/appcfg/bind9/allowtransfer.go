package bind9config

import "slices"

// Checks if the zone transfer is disabled.
func (at *AllowTransfer) IsDisabled() bool {
	// By default, the transfer is disabled. It is also disabled when it is none.
	// If any of the elements is not none, the transfer is enabled.
	return len(at.AdressMatchList.Elements) == 0 || !slices.ContainsFunc(at.AdressMatchList.Elements, func(ame *AddressMatchListElement) bool {
		return ame.ACLName != "none"
	})
}
