package keaconfig

import dhcpmodel "isc.org/stork/datamodel/dhcp"

// DHCP option type enum, as defined in Kea.
type DHCPOptionType = string

// Valid DHCP option types.
const (
	EmptyOption       DHCPOptionType = "empty"
	StringOption      DHCPOptionType = "string"
	BoolOption        DHCPOptionType = "bool"
	Uint8Option       DHCPOptionType = "uint8"
	Uint16Option      DHCPOptionType = "uint16"
	Uint32Option      DHCPOptionType = "uint32"
	Int8Option        DHCPOptionType = "int8"
	Int16Option       DHCPOptionType = "int16"
	Int32Option       DHCPOptionType = "int32"
	IPv4AddressOption DHCPOptionType = "ipv4-address"
	IPv6AddressOption DHCPOptionType = "ipv6-address"
	IPv6PrefixOption  DHCPOptionType = "ipv6-prefix"
	PsidOption        DHCPOptionType = "psid"
	FqdnOption        DHCPOptionType = "fqdn"
	TupleOption       DHCPOptionType = "tuple"
	RecordOption      DHCPOptionType = "record"
)

// DHCP option definition in the format used by Kea.
type dhcpOptionDefinition struct {
	Array       bool             `json:"array,omitempty"`
	Code        uint16           `json:"code"`
	Encapsulate string           `json:"encapsulate"`
	Name        string           `json:"name"`
	RecordTypes []DHCPOptionType `json:"record-types"`
	Space       string           `json:"space"`
	OptionType  DHCPOptionType   `json:"type"`
}

// DHCP option definition interface.
type DHCPOptionDefinition interface {
	GetArray() bool
	GetCode() uint16
	GetEncapsulate() string
	GetName() string
	GetRecordTypes() []DHCPOptionType
	GetSpace() string
	GetType() DHCPOptionType
}

// An interface to a structure providing option definition lookup capabilities.
type DHCPOptionDefinitionLookup interface {
	// Checks if a definition of the specified option exists for the
	// given daemon.
	DefinitionExists(int64, dhcpmodel.DHCPOptionAccessor) bool
	// Searches for an option definition for the specified daemon ID and option value.
	Find(int64, dhcpmodel.DHCPOptionAccessor) DHCPOptionDefinition
}

// Checks if the option is an array (has an array of option fields).
func (def dhcpOptionDefinition) GetArray() bool {
	return def.Array
}

// Returns option code.
func (def dhcpOptionDefinition) GetCode() uint16 {
	return def.Code
}

// Returns option space encapsulated by the option.
func (def dhcpOptionDefinition) GetEncapsulate() string {
	return def.Encapsulate
}

// Returns option name.
func (def dhcpOptionDefinition) GetName() string {
	return def.Name
}

// Returns record types (when an option is a record of different fields).
func (def dhcpOptionDefinition) GetRecordTypes() []DHCPOptionType {
	return def.RecordTypes
}

// Returns option space.
func (def dhcpOptionDefinition) GetSpace() string {
	return def.Space
}

// Returns option type.
func (def dhcpOptionDefinition) GetType() DHCPOptionType {
	return def.OptionType
}

// Given the option definition, find field type at specified position.
// First option field has position 0. If the position is out of bounds,
// the second returned parameter is false and the option field type
// is empty. For an empty option this function always returns false and
// empty option field type.
func GetDHCPOptionDefinitionFieldType(def DHCPOptionDefinition, position int) (dhcpmodel.DHCPOptionFieldType, bool) {
	switch def.GetType() {
	case EmptyOption:
		return "", false
	case RecordOption:
		recordTypes := def.GetRecordTypes()
		// Empty record types is theoretically impossible because Kea doesn't
		// allow it. However, let's be safe and check because it may cause
		// division by 0. Also, if it is not an array and the position is
		// out of the record boundaries, return false.
		if len(recordTypes) == 0 || (!def.GetArray() && (position > len(recordTypes)-1)) {
			return "", false
		}
		recordPosition := position % len(recordTypes)
		return recordTypes[recordPosition], true
	default:
		if position > 0 && !def.GetArray() {
			return "", false
		}
		return def.GetType(), true
	}
}
