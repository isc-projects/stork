package keaconfig

import (
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	errors "github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// DHCP option space (one of dhcp4 or dhcp6).
type DHCPOptionSpace = string

// Top level DHCP option spaces.
const (
	DHCPv4OptionSpace DHCPOptionSpace = "dhcp4"
	DHCPv6OptionSpace DHCPOptionSpace = "dhcp6"
)

// Type of a DHCP option field.
type DHCPOptionFieldType = string

// Supported types of DHCP option fields.
const (
	HexBytesField    DHCPOptionFieldType = "hex-bytes"
	StringField      DHCPOptionFieldType = "string"
	BoolField        DHCPOptionFieldType = "bool"
	Uint8Field       DHCPOptionFieldType = "uint8"
	Uint16Field      DHCPOptionFieldType = "uint16"
	Uint32Field      DHCPOptionFieldType = "uint32"
	IPv4AddressField DHCPOptionFieldType = "ipv4-address"
	IPv6AddressField DHCPOptionFieldType = "ipv6-address"
	IPv6PrefixField  DHCPOptionFieldType = "ipv6-prefix"
	PsidField        DHCPOptionFieldType = "psid"
	FqdnField        DHCPOptionFieldType = "fqdn"
)

// An interface to an option field. It returns an option field type
// and its value(s). Database model representing DHCP option fields
// implements this interface.
type DHCPOptionField interface {
	// Returns option field type.
	GetFieldType() string
	// Returns option field value(s).
	GetValues() []any
}

// An interface to a DHCP option. Database model representing DHCP options
// implements this interface.
type DHCPOption interface {
	// Returns a boolean flag indicating if the option should be
	// always returned, regardless whether it is requested or not.
	IsAlwaysSend() bool
	// Returns option code.
	GetCode() uint16
	// Returns encapsulated option space name.
	GetEncapsulate() string
	// Returns option fields.
	GetFields() []DHCPOptionField
	// Returns option name.
	GetName() string
	// Returns option space.
	GetSpace() string
	// Returns the universe (i.e., IPv4 or IPv6).
	GetUniverse() storkutil.IPType
}

// Represents a DHCP option in the format used by Kea (i.e., an item of the
// option-data list).
type SingleOptionData struct {
	AlwaysSend bool   `mapstructure:"always-send" json:"always-send,omitempty"`
	Code       uint16 `mapstructure:"code" json:"code,omitempty"`
	CSVFormat  bool   `mapstructure:"csv-format" json:"csv-format"`
	Data       string `mapstructure:"data" json:"data,omitempty"`
	Name       string `mapstructure:"name" json:"name,omitempty"`
	Space      string `mapstructure:"space" json:"space,omitempty"`
}

// Creates a SingleOptionData instance from the DHCP option model used
// by Stork (e.g., from an option held in the Stork database). If the
// option has a definition, it uses the Kea's csv-format setting and
// specifies the option value as a comma separated list. Otherwise, it
// converts the option fields to a hex form and sets the csv-format to
// false. The lookup interface must not be nil.
func CreateSingleOptionData(daemonID int64, lookup DHCPOptionDefinitionLookup, option DHCPOption) (*SingleOptionData, error) {
	// Create Kea representation of the option. Set csv-format to
	// true for all options for which the definitions are known.
	data := &SingleOptionData{
		AlwaysSend: option.IsAlwaysSend(),
		Code:       option.GetCode(),
		CSVFormat:  lookup.DefinitionExists(daemonID, option),
		Name:       option.GetName(),
		Space:      option.GetSpace(),
	}
	// Convert option fields depending on the csv-format setting.
	converted := []string{}
	for _, field := range option.GetFields() {
		var (
			value string
			err   error
		)
		switch field.GetFieldType() {
		case HexBytesField:
			value, err = convertHexBytesField(field)
		case StringField:
			value, err = convertStringField(field, data.CSVFormat)
		case BoolField:
			value, err = convertBoolField(field, data.CSVFormat)
		case Uint8Field, Uint16Field, Uint32Field:
			value, err = convertUintField(field, data.CSVFormat)
		case IPv4AddressField:
			value, err = convertIPv4AddressField(field, data.CSVFormat)
		case IPv6AddressField:
			value, err = convertIPv6AddressField(field, data.CSVFormat)
		case IPv6PrefixField:
			value, err = convertIPv6PrefixField(field, data.CSVFormat)
		case PsidField:
			value, err = convertPsidField(field, data.CSVFormat)
		case FqdnField:
			value, err = convertFqdnField(field, data.CSVFormat)
		default:
			err = errors.Errorf("unsupported option field type %s", field.GetFieldType())
		}
		if err != nil {
			return nil, err
		}
		// The value can be a string with the option field value or a string
		// of hexadecimal digits representing the value.
		converted = append(converted, value)
	}
	if data.CSVFormat {
		// Use comma separated values.
		data.Data = strings.Join(converted, ",")
	} else {
		// The option is specified as a string of hexadecimal digits. Let's
		// just concatenate all option fields into a single string.
		data.Data = strings.Join(converted, "")
	}
	return data, nil
}

// Converts a hex-bytes option field from Stork to Kea format. The conversion
// is the same regardless of the csv-format setting. It removes the colons
// and spaces if they are used as separators.
func convertHexBytesField(field DHCPOptionField) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in hex-bytes option field, have %d", len(values))
	}
	output, ok := values[0].(string)
	if !ok {
		return "", errors.New("hex-bytes option field is not a string value")
	}
	if len(output) == 0 {
		return "", errors.New("hex-bytes option field value must not be empty")
	}
	for _, sep := range []string{":", " "} {
		output = strings.ReplaceAll(output, sep, "")
	}
	if _, err := hex.DecodeString(output); err != nil {
		return "", errors.Wrap(err, "option field is not a valid string of hexadecimal digits")
	}
	return output, nil
}

// Converts a string option field from Stork to Kea format. It expects that
// there is one option field value and that this value is a string. If the
// textFormat is set to true, it returns the original string. Otherwise, it
// converts the string to the hex format.
func convertStringField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in string option field, have %d", len(values))
	}
	output, ok := values[0].(string)
	if !ok {
		return "", errors.New("string option field is not a string value")
	}
	if len(output) == 0 {
		return "", errors.New("string option field value must not be empty")
	}
	if textFormat {
		return output, nil
	}
	return storkutil.BytesToHex([]byte(output)), nil
}

// Converts a boolean option field from Stork to Kea format. It expects that
// there is one option field value and that this value is a boolean. If the
// textFormat is set to true, it returns "true" or "false". Otherwise, it
// returns "01" or "00" (i.e., hex format).
func convertBoolField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in bool option field, have %d", len(values))
	}
	b, ok := values[0].(bool)
	if !ok {
		return "", errors.New("bool option field contains invalid value")
	}
	if textFormat {
		return fmt.Sprintf("%t", b), nil
	}
	if b {
		return "01", nil
	}
	return "00", nil
}

// Converts an uint8, uint16 or uin32 option field from Stork to Kea format.
// It expects that there is one option field value and that this value is a
// number. If the textFormat is set to true, it returns the value converted
// to a string. Otherwise, it converts the value to the hex format.
func convertUintField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in uint option field, have %d", len(values))
	}
	if !storkutil.IsWholeNumber(values[0]) {
		return "", errors.New("uint option field value is not a valid number")
	}
	value := reflect.ValueOf(values[0])
	ivalue := value.Convert(reflect.TypeOf((*uint64)(nil)).Elem())
	switch field.GetFieldType() {
	case Uint8Field:
		if ivalue.Uint() > math.MaxUint8 {
			return "", errors.Errorf("uint8 option field value must not be greater than math.MaxUint8")
		}
		if !textFormat {
			return fmt.Sprintf("%02X", ivalue.Uint()), nil
		}
	case Uint16Field:
		if ivalue.Uint() > math.MaxUint16 {
			return "", errors.Errorf("uint16 option field value must not be greater than %d", math.MaxUint16)
		}
		if !textFormat {
			return fmt.Sprintf("%04X", ivalue.Uint()), nil
		}
	case Uint32Field:
		if ivalue.Uint() > math.MaxUint32 {
			return "", errors.Errorf("uint32 option field value must not be greater than %d", math.MaxUint32)
		}
		if !textFormat {
			return fmt.Sprintf("%08X", ivalue.Uint()), nil
		}
	}
	return fmt.Sprintf("%d", values[0]), nil
}

// Converts an IPv4 address option field from Stork to Kea format. It expects
// that there is one option field value and that this value is a string with
// a valid IPv4 address. If the textFormat is set to true, it returns the
// original string. Otherwise, it converts the IPv4 address to the hex format.
func convertIPv4AddressField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in IPv4 address option field, have %d", len(values))
	}
	ip, ok := values[0].(string)
	if !ok {
		return "", errors.New("IPv4 address option field is not a string value")
	}
	parsed := storkutil.ParseIP(ip)
	if parsed == nil || parsed.Protocol != storkutil.IPv4 {
		return "", errors.New("IPv4 address option field contains invalid value")
	}
	if textFormat {
		return ip, nil
	}
	return storkutil.BytesToHex(parsed.IP.To4()), nil
}

// Converts an IPv6 address option field from Stork to Kea format. It expects
// that there is one option field value and that this value is a string with
// a valid IPv6 address. If the textFormat is set to true, it returns the
// original string. Otherwise, it converts the IPv6 address to the hex format.
func convertIPv6AddressField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in IPv6 address option field, have %d", len(values))
	}
	ip, ok := values[0].(string)
	if !ok {
		return "", errors.New("IPv6 address option field is not a string value")
	}
	parsed := storkutil.ParseIP(ip)
	if parsed == nil || parsed.Protocol != storkutil.IPv6 {
		return "", errors.New("IPv6 address option field contains invalid value")
	}
	if textFormat {
		return ip, nil
	}
	return storkutil.BytesToHex(parsed.IP), nil
}

// Converts an IPv6 delegated prefix option field from Stork to Kea format.
// It expects that there are two option field values, one is a string with a
// prefix and the second one is a prefix length. If the textFormat is set to
// true, it returns the prefix in a prefix/length format. Otherwise, it
// converts the values to the hex format and returns them concatenated.
func convertIPv6PrefixField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 2 {
		return "", errors.Errorf("IPv6 prefix option field must contain two values, it has %d", len(values))
	}
	ip, ok := values[0].(string)
	if !ok {
		return "", errors.New("IPv6 prefix in the option field is not a string value")
	}
	if !storkutil.IsWholeNumber(values[1]) {
		return "", errors.New("IPv6 prefix length in the option field is not a number")
	}
	prefixLen := reflect.ValueOf(values[1]).Convert(reflect.TypeOf((*int64)(nil)).Elem())
	if prefixLen.Int() <= 0 || prefixLen.Int() > 128 {
		return "", errors.New("IPv6 prefix length must be a positive number not greater than 128")
	}
	parsed := storkutil.ParseIP(ip)
	if parsed == nil || parsed.Protocol != storkutil.IPv6 {
		return "", errors.New("IPv6 prefix option field contains invalid value")
	}
	if textFormat {
		return fmt.Sprintf("%s/%d", values[0], values[1]), nil
	}
	return fmt.Sprintf("%s%02X", storkutil.BytesToHex(parsed.IP), prefixLen.Int()), nil
}

// Converts a PSID option field from Stork to Kea format. It expects that
// there are two option field values, one is a uint16 value specifying PSID,
// and the second one is a uint8 value specifying PSID length. If the
// textFormat is set to true, it returns the option field value in the
// PSID/PSIDLen format. Otherwise, it converts the option fields to the hex
// format and returns them concatenated.
func convertPsidField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 2 {
		return "", errors.Errorf("psid option field must contain two values, it has %d", len(values))
	}
	for _, v := range values {
		if !storkutil.IsWholeNumber(v) {
			return "", errors.New("values in the psid option field must be numbers")
		}
	}
	psid := reflect.ValueOf(values[0]).Convert(reflect.TypeOf((*int64)(nil)).Elem())
	if psid.Int() <= 0 || psid.Int() > math.MaxUint16 {
		return "", errors.Errorf("psid value must be a positive number not greater than %d", math.MaxUint16)
	}
	psidLength := reflect.ValueOf(values[1]).Convert(reflect.TypeOf((*int64)(nil)).Elem())
	if psidLength.Int() <= 0 || psidLength.Int() > math.MaxUint8 {
		return "", errors.Errorf("psid length must be a positive number must not be greater than %d", math.MaxUint8)
	}
	if textFormat {
		return fmt.Sprintf("%d/%d", values[0], values[1]), nil
	}
	return fmt.Sprintf("%04X%02X", psid.Int(), psidLength.Int()), nil
}

// Converts an FQDN option field from Stork to Kea format. It expects that
// there is one option field value and that this value is a string with
// a valid FQDN. If the textFormat is set to true, it returns the original
// string. Otherwise, it converts the FQDN to the hex format. A partial
// FQDN should lack a terminating dot. A full FQDN should include a terminating
// dot.
func convertFqdnField(field DHCPOptionField, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in FQDN option field, have %d", len(values))
	}
	value, ok := values[0].(string)
	if !ok {
		return "", errors.New("FQDN option field is not a string value")
	}
	fqdn, err := storkutil.ParseFqdn(value)
	if err != nil {
		return "", err
	}
	if textFormat {
		return value, nil
	}
	fqdnBytes, err := fqdn.ToBytes()
	if err != nil {
		return "", errors.Errorf("failed to parse FQDN option field: %s", err)
	}
	return storkutil.BytesToHex(fqdnBytes), nil
}

// Represents a DHCP option field and implements DHCPOptionField interface. It
// is returned by the CreateDHCPOption function.
type dhcpOptionField struct {
	FieldType string
	Values    []any
}

// Represents a DHCP option and implements DHCPOption interface. It is returned
// by the CreateDHCPOption function.
type dhcpOption struct {
	AlwaysSend  bool
	Code        uint16
	Encapsulate string
	Fields      []DHCPOptionField
	Name        string
	Space       string
	Universe    storkutil.IPType
}

// Returns option field type.
func (field dhcpOptionField) GetFieldType() string {
	return field.FieldType
}

// Returns option field values.
func (field dhcpOptionField) GetValues() []any {
	return field.Values
}

// Checks if the option is always returned to a DHCP client, regardless
// if the client has requested it or not.
func (option dhcpOption) IsAlwaysSend() bool {
	return option.AlwaysSend
}

// Returns option code.
func (option dhcpOption) GetCode() uint16 {
	return option.Code
}

// Returns an encapsulated option space name.
func (option dhcpOption) GetEncapsulate() string {
	return option.Encapsulate
}

// Returns option fields belonging to the option.
func (option dhcpOption) GetFields() (returnedFields []DHCPOptionField) {
	return option.Fields
}

// Returns option name.
func (option dhcpOption) GetName() string {
	return option.Name
}

// Returns option universe (i.e., IPv4 or IPv6).
func (option dhcpOption) GetUniverse() storkutil.IPType {
	return option.Universe
}

// Returns option space name.
func (option dhcpOption) GetSpace() string {
	return option.Space
}

// Tries to infer option field type from its value and converts it to the
// format used in Stork. The resulting format comprises an option field type
// and the corresponding field value(s). It has some limitations:
//
// - It is unable to recognize an exact integer type, therefore it returns
//   all numbers as uint32 fields.
// - It is unable to differentiate between partial FQDN and a regular string,
//   therefore it returns string field for partial FQDNs.
//
// The function is used temporarily until we add support for Kea option
// definitions. However, even then, this function may be useful in cases when
// the definition is not available for any reason.
func inferDHCPOptionField(value string) dhcpOptionField {
	var field dhcpOptionField

	// Is it a bool value?
	if bv, err := strconv.ParseBool(value); err == nil {
		field = dhcpOptionField{
			FieldType: BoolField,
			Values:    []any{bv},
		}
		return field
	}
	// Is it a number?
	if iv, err := strconv.ParseUint(value, 10, 32); err == nil {
		field = dhcpOptionField{
			FieldType: Uint32Field,
			Values:    []any{iv},
		}
		return field
	}
	// Is it an IP address or prefix?
	if ip := storkutil.ParseIP(value); ip != nil {
		switch ip.Protocol {
		// Is it an IPv4 address?
		case storkutil.IPv4:
			field = dhcpOptionField{
				FieldType: IPv4AddressField,
				Values:    []any{value},
			}
			return field
		// Is it an IPv6 address or prefix?
		case storkutil.IPv6:
			// Is it a prefix?
			if ip.Prefix {
				field = dhcpOptionField{
					FieldType: IPv6PrefixField,
					Values: []any{
						ip.NetworkPrefix,
						ip.PrefixLength,
					},
				}
			} else {
				// Is it an IPv6 address?
				field = dhcpOptionField{
					FieldType: IPv6AddressField,
					Values:    []any{value},
				}
			}
			return field
		}
	}
	// Is it an FQDN?
	if fqdn, err := storkutil.ParseFqdn(value); err == nil {
		if !fqdn.IsPartial() {
			field = dhcpOptionField{
				FieldType: FqdnField,
				Values:    []any{value},
			}
			return field
		}
	}
	// Is it PSID?
	pv := strings.Split(value, "/")
	if len(pv) == 2 {
		if psid, err := strconv.ParseUint(pv[0], 10, 16); err == nil {
			if psidLen, err := strconv.ParseUint(pv[1], 10, 8); err == nil {
				field = dhcpOptionField{
					FieldType: PsidField,
					Values:    []any{psid, psidLen},
				}
				return field
			}
		}
	}
	// It must be a string.
	field = dhcpOptionField{
		FieldType: StringField,
		Values:    []any{value},
	}
	return field
}

// Creates an instance of a DHCP option in Stork from the option representation
// in Kea. This function does not recognize encapsulated option space because
// it is unavailable in the returned option data. To recognize the encapsulated
// option space we need to add support for option definitions.
func CreateDHCPOption(optionData SingleOptionData, universe storkutil.IPType) DHCPOption {
	option := dhcpOption{
		AlwaysSend: optionData.AlwaysSend,
		Code:       optionData.Code,
		Name:       optionData.Name,
		Space:      optionData.Space,
		Universe:   universe,
	}
	data := strings.TrimSpace(optionData.Data)

	// Option encapsulation.
	switch option.Space {
	case DHCPv4OptionSpace, DHCPv6OptionSpace:
		option.Encapsulate = fmt.Sprintf("option-%d", option.Code)
	default:
		option.Encapsulate = fmt.Sprintf("%s.%d", option.Space, option.Code)
	}

	// There is nothing to do if the option is empty.
	if len(data) == 0 {
		return option
	}

	// Option data specified as comma separated values.
	if optionData.CSVFormat {
		values := strings.Split(data, ",")
		for _, raw := range values {
			v := strings.TrimSpace(raw)
			field := inferDHCPOptionField(v)
			option.Fields = append(option.Fields, field)
		}
		return option
	}

	// If the csv-format is false the option payload is specified using a string
	// of hexadecimal digits. Sanitize colons and whitespaces.
	data = strings.ReplaceAll(strings.ReplaceAll(data, " ", ""), ":", "")
	field := dhcpOptionField{
		FieldType: HexBytesField,
		Values:    []any{data},
	}
	option.Fields = append(option.Fields, field)

	return option
}
