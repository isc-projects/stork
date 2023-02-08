package dhcpmodel

// Type of a DHCP option field.
type DHCPOptionFieldType = string

// Supported types of DHCP option fields.
const (
	BinaryField      DHCPOptionFieldType = "binary"
	StringField      DHCPOptionFieldType = "string"
	BoolField        DHCPOptionFieldType = "bool"
	Uint8Field       DHCPOptionFieldType = "uint8"
	Uint16Field      DHCPOptionFieldType = "uint16"
	Uint32Field      DHCPOptionFieldType = "uint32"
	Int8Field        DHCPOptionFieldType = "int8"
	Int16Field       DHCPOptionFieldType = "int16"
	Int32Field       DHCPOptionFieldType = "int32"
	IPv4AddressField DHCPOptionFieldType = "ipv4-address"
	IPv6AddressField DHCPOptionFieldType = "ipv6-address"
	IPv6PrefixField  DHCPOptionFieldType = "ipv6-prefix"
	PsidField        DHCPOptionFieldType = "psid"
	FqdnField        DHCPOptionFieldType = "fqdn"
)

// An interface to an option field. It returns an option field type
// and its value(s). Database model representing DHCP option fields
// implements this interface.
type DHCPOptionFieldAccessor interface {
	// Returns option field type.
	GetFieldType() string
	// Returns option field value(s).
	GetValues() []any
}
