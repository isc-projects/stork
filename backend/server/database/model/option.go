package dbmodel

import (
	keaconfig "isc.org/stork/appcfg/kea"
	storkutil "isc.org/stork/util"
)

// Represents a DHCP option field.
type DHCPOptionField struct {
	FieldType string
	Values    []interface{}
}

// Represents a DHCP option.
type DHCPOption struct {
	AlwaysSend  bool
	Code        uint16
	Encapsulate string
	Fields      []DHCPOptionField
	Name        string
	Space       string
	Universe    storkutil.IPType
}

// Returns option field type.
func (field DHCPOptionField) GetFieldType() string {
	return field.FieldType
}

// Returns option field values.
func (field DHCPOptionField) GetValues() []interface{} {
	return field.Values
}

// Checks if the option is always returned to a DHCP client, regardless
// if the client has requested it or not.
func (option DHCPOption) IsAlwaysSend() bool {
	return option.AlwaysSend
}

// Returns option code.
func (option DHCPOption) GetCode() uint16 {
	return option.Code
}

// Returns an encapsulated option space name.
func (option DHCPOption) GetEncapsulate() string {
	return option.Encapsulate
}

// Returns option fields belonging to the option.
func (option DHCPOption) GetFields() (returnedFields []keaconfig.DHCPOptionField) {
	for _, field := range option.Fields {
		returnedFields = append(returnedFields, field)
	}
	return returnedFields
}

// Returns option name.
func (option DHCPOption) GetName() string {
	return option.Name
}

// Returns option universe (i.e., IPv4 or IPv6).
func (option DHCPOption) GetUniverse() storkutil.IPType {
	return option.Universe
}

// Returns option space name.
func (option DHCPOption) GetSpace() string {
	return option.Space
}
