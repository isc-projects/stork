package bind9config

// Checks if the address match list excludes the specified IP address.
func (aml *AddressMatchList) ExcludesIPAddress(ipAddress string) bool {
	for _, element := range aml.Elements {
		if (element.IPAddress == ipAddress && element.Negation) ||
			(element.ACLName == "none" && !element.Negation) ||
			(element.ACLName == "any" && element.Negation) {
			return true
		}
	}
	return false
}
