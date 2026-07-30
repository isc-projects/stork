package bind9config

import (
	"maps"
	"slices"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// The maximum recursion level allowed when matching in the address match list.
const maxAddressMatchListRecursionLevel = 10

// An interface exposing functions to access the global BIND 9 configuration
// required by the address match list.
type AddressMatchListGlobalConfigAccessor interface {
	GetACL(aclName string) *ACL
	GetKey(keyID string) *Key
}

// A type of the function checking if the IP address belongs to the local network.
// It is used within this package in the match function signature.
type addressMatchFn func(ipAddress string) bool

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

// Checks if the specified IP address and key are allowed by the address match list.
// It recursively matches the IP addresses in the embedded ACLs and match lists.
// It returns an error if the recursion level exceeds the maximum allowed level.
// The IP address is allowed when it has a positive match. It is denied on
// first negative match (regardless if it later has positive matches). It is
// also denied when it has no matches (either negative or positive).
func (aml *AddressMatchList) IsIPAddressAndKeyAllowed(globalConfig AddressMatchListGlobalConfigAccessor, ipAddress, keyID string) (bool, error) {
	match, negative, err := aml.match(0, globalConfig, ipAddress, keyID, storkutil.IsHostIPAddress, storkutil.IsIPAddressInHostNetwork)
	if err != nil {
		return false, err
	}
	return match && !negative, nil
}

// Matches the IP address or key ID against the address match list elements.
// It recursively matches the IP addresses or key IDs in the embedded ACLs and
// match lists. This function takes recursion level as an argument. It may
// recursively call itself increasing the recursion level. It is called internally
// by the IsIPAddressAllowed and IsKeyAllowed functions. It returns a boolean
// indicating if the IP address or key ID is matched, a boolean indicating if the
// match is negative (!), and an error if the recursion level exceeds the maximum
// allowed level.
func (aml *AddressMatchList) match(level int, globalConfig AddressMatchListGlobalConfigAccessor, ipAddress, keyID string, localAddressMatchFn addressMatchFn, localNetworkMatchFn addressMatchFn) (bool, bool, error) {
	if level > maxAddressMatchListRecursionLevel {
		// Too much recursion in the BIND 9 configuration.
		return false, false, errors.Errorf("address match list recursion level exceeded: %d", level)
	}
	for _, element := range aml.Elements {
		var (
			match    bool
			negative bool
			err      error
		)
		switch {
		case element.IPAddressOrACLName == "none":
			// None of the IP addresses match the none ACL.
			match = false
			negative = element.Negation
		case element.IPAddressOrACLName == "any":
			// All IP addresses match the any ACL.
			match = true
			negative = element.Negation
		case element.IPAddressOrACLName == "localhost":
			// Match against all IP addresses assigned to the host interfaces.
			match = localAddressMatchFn != nil && localAddressMatchFn(ipAddress)
			negative = element.Negation
		case element.IPAddressOrACLName == "localnets":
			// Match against all IP addresses belonging to the host networks.
			match = localNetworkMatchFn != nil && localNetworkMatchFn(ipAddress)
			negative = element.Negation
		case element.KeyID != "" && keyID != "":
			// The current element is a key. Let's simply compare key IDs.
			match = element.KeyID == keyID
			negative = element.Negation
		case element.IPAddressOrACLName != "":
			if element.IsIPAddress() {
				if ipAddress != "" {
					// An IP address specified in the address match list.
					// Compare it withe IP address in the argument.
					match = element.IPAddressOrACLName == ipAddress
					negative = element.Negation
				}
				break
			}
			// It seems that the current element is an ACL name.
			// Let's try to get the ACL from the global config.
			acl := globalConfig.GetACL(element.IPAddressOrACLName)
			if acl != nil && acl.AddressMatchList != nil {
				// ACL exists. Let's look into its address match list and try
				// matching the IP address against it.
				match, negative, err = acl.AddressMatchList.match(level+1, globalConfig, ipAddress, keyID, localAddressMatchFn, localNetworkMatchFn)
				if err != nil {
					// Too-much recursion.
					return false, false, err
				}
			}
			if match && element.Negation {
				if negative {
					// We have a negative match in the referenced ACL. The ACL is negated, so
					// in this case BIND 9 moves on to next element to look for a positive match.
					// Therefore, in this case we clear the match flag to allow the process to
					// continue. See: https://web.mit.edu/darwin/src/modules/bind/bind/doc/html/address_list.html
					// for more details how BIND 9 handles outer negations in the embedded ACLs.
					match = false
				}
				// Make sure that the negative flag is set since the element is negated.
				negative = true
			}
		case element.AddressMatchList != nil:
			// The current element is an embedded address match list.
			// Let's try to match the IP address against it.
			match, negative, err = element.AddressMatchList.match(level+1, globalConfig, ipAddress, keyID, localAddressMatchFn, localNetworkMatchFn)
			if err != nil {
				return false, false, err
			}
			if match && element.Negation {
				if negative {
					// We have a negative match in the embedded match list. The match list is negated,
					// so in this case BIND 9 moves on to next element to look for a positive match.
					// Therefore, in this case we clear the match flag to allow the process to
					// continue. See: https://web.mit.edu/darwin/src/modules/bind/bind/doc/html/address_list.html
					// for more details how BIND 9 handles outer negations in the embedded match lists.
					match = false
				}
				negative = true
			}
		}
		if !match {
			// If there is no match (either positive or negative), continue
			// with the next element.
			continue
		}
		// There was a match - positive or negative. Return the result.
		return match, negative, nil
	}
	return false, false, nil
}

// Returns all keys in the address match list. It uses the interface to the global
// configuration to get the actual details of the keys.
func (aml *AddressMatchList) GetKeys(globalConfig AddressMatchListGlobalConfigAccessor) ([]*Key, error) {
	var keys []*Key
	ids, err := aml.getKeyIDs(0, globalConfig)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if key := globalConfig.GetKey(id); key != nil {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// Gets all key IDs in the address match list. It takes recursion level as an
// argument. It may recursively call itself increasing the recursion level.
// It is called internally by the GetFirstKeyAllowed function. It returns
// a slice of key IDs and an error if the recursion level exceeds the maximum
// allowed level.
func (aml *AddressMatchList) getKeyIDs(level int, globalConfig AddressMatchListGlobalConfigAccessor) ([]string, error) {
	if level > maxAddressMatchListRecursionLevel {
		// Too much recursion in the BIND 9 configuration.
		return nil, errors.Errorf("address match list recursion level exceeded: %d", level)
	}
	keyIDs := make(map[string]struct{})
	for _, element := range aml.Elements {
		switch {
		case element.KeyID != "":
			// The current element is a key. Let's add it to the list.
			keyIDs[element.KeyID] = struct{}{}
		case element.IPAddressOrACLName != "" && !element.IsIPAddress():
			// The current element is ACL name. Let's try to get the ACL
			// from the global config.
			if acl := globalConfig.GetACL(element.IPAddressOrACLName); acl != nil && acl.AddressMatchList != nil {
				// ACL exists. Let's look into its address match list and try
				// to get the key IDs.
				ids, err := acl.AddressMatchList.getKeyIDs(level+1, globalConfig)
				if err != nil {
					return nil, err
				}
				for _, id := range ids {
					keyIDs[id] = struct{}{}
				}
			}
		case element.AddressMatchList != nil:
			// The current element is an embedded address match list.
			// Let's look into it and try to get the key IDs.
			ids, err := element.AddressMatchList.getKeyIDs(level+1, globalConfig)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				keyIDs[id] = struct{}{}
			}
		}
	}
	return slices.Collect(maps.Keys(keyIDs)), nil
}
