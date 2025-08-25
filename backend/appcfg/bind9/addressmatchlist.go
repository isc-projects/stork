package bind9config

// Checks if the address match list excludes the specified IP address.
func (aml *AddressMatchList) ExcludesIPAddress(ipAddress string) bool {
	for _, element := range aml.Elements {
		if (element.IPAddressOrACLName == ipAddress && element.Negation) ||
			(element.IPAddressOrACLName == "none" && !element.Negation) ||
			(element.IPAddressOrACLName == "any" && element.Negation) {
			return true
		}
	}
	return false
}
