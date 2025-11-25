package bind9config

// AddressMatchList is the list of address match list elements between curly braces.
// The address match list elements include but are not limited to: IP addresses,
// keys, or ACLs. The elements may also contain a negation sign. It is used to
// exclude certain clients from the ACLs. The address match list has the following
// format:
//
//	[ ! ] ( <ip_address> | <netprefix> | key <server_key> | <acl_name> | { address_match_list } )
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#term-address_match_element.
type AddressMatchList struct {
	Elements []*AddressMatchListElement `parser:"( @@ ';'* )*"`
}

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
