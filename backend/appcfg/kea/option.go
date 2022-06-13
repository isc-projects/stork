package keaconfig

import (
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"strings"

	errors "github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

// Top level DHCP option spaces.
const (
	DHCPv4OptionSpace = "dhcp4"
	DHCPv6OptionSpace = "dhcp6"
)

// Types of DHCP option fields.
const (
	HexBytesField    string = "hex-bytes"
	StringField      string = "string"
	BoolField        string = "bool"
	Uint8Field       string = "uint8"
	Uint16Field      string = "uint16"
	Uint32Field      string = "uint32"
	IPv4AddressField string = "ipv4-address"
	IPv6AddressField string = "ipv6-address"
	IPv6PrefixField  string = "ipv6-prefix"
	PsidField        string = "psid"
	FqdnField        string = "fqdn"
)

// An interface to an option field. It returns an option field type
// and its value(s). Database model representing DHCP option fields
// implements this interface.
type DHCPOptionField interface {
	// Returns option field type.
	GetFieldType() string
	// Returns option field value(s).
	GetValues() []interface{}
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

// An interface to a structure providing option definition lookup capabilities.
type DHCPOptionDefinitionLookup interface {
	// Checks if a definition of the specified option exists for the
	// given daemon.
	DefinitionExists(int64, DHCPOption) bool
}

// Represents a DHCP option in the format used by Kea (i.e., an item of the
// option-data list).
type SingleOptionData struct {
	AlwaysSend bool   `mapstructure:"always-send" json:"always-send,omitempty"`
	Code       uint16 `mapstructure:"code" json:"code,omitempty"`
	CSVFormat  bool   `mapstructure:"csv-format" json:"csv-format,omitempty"`
	Data       string `mapstructure:"data" json:"data,omitempty"`
	Name       string `mapstructure:"name" json:"name,omitempty"`
	Space      string `mapstructure:"space" json:"space,omitempty"`
}

// Creates a SingleOptionData instance from the DHCP option model used
// by Stork (e.g., from an option held in the Stork database). If the
// option has a definition, it uses the Kea's csv-format setting and
// specifies the option value as a comma separated list. Otherwise, it
// converts the option fields to a hex form and sets the csv-format to
// false.
func CreateSingleOptionData(daemonID int64, lookup DHCPOptionDefinitionLookup, option DHCPOption) (*SingleOptionData, error) {
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
		return "", errors.New("multiple values in hex-bytes option field")
	}
	output, ok := values[0].(string)
	if !ok {
		return "", errors.New("hex-bytes option field is not a string value")
	}
	for _, sep := range []string{":", " "} {
		output = strings.ReplaceAll(output, sep, "")
	}
	if _, err := hex.DecodeString(output); err != nil {
		return "", errors.New("option field is not a valid string of hexadecimal digits")
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
		return "", errors.New("multiple values in string option field")
	}
	output, ok := values[0].(string)
	if !ok {
		return "", errors.New("string option field is not a string value")
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
		return "", errors.New("multiple values in the bool option field")
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
		return "", errors.New("multiple values in the uint option field")
	}
	value := reflect.ValueOf(values[0])
	if !value.CanConvert(reflect.TypeOf((*uint64)(nil)).Elem()) {
		return "", errors.New("uint option field value is not a valid number")
	}
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
		return "", errors.New("multiple values in the IPv4 address option field")
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
		return "", errors.New("multiple values in the IPv6 address option field")
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
		return "", errors.New("IPv6 prefix option field must contain two values")
	}
	ip, ok := values[0].(string)
	if !ok {
		return "", errors.New("IPv6 prefix in the option field is not a string value")
	}
	if !reflect.TypeOf(values[1]).ConvertibleTo(reflect.TypeOf((*uint64)(nil)).Elem()) {
		return "", errors.New("IPv6 prefix length in the option field is not a number")
	}
	len := reflect.ValueOf(values[1])
	if len.Int() > 128 {
		return "", errors.New("IPv6 prefix length must not be greater than 128")
	}
	parsed := storkutil.ParseIP(ip)
	if parsed == nil || parsed.Protocol != storkutil.IPv6 {
		return "", errors.New("IPv6 prefix option field contains invalid value")
	}
	if textFormat {
		return fmt.Sprintf("%s/%d", values[0], values[1]), nil
	}
	return fmt.Sprintf("%s%02X", storkutil.BytesToHex(parsed.IP), len.Int()), nil
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
		return "", errors.New("psid option field must contain two values")
	}
	for _, v := range values {
		if !reflect.TypeOf(v).ConvertibleTo(reflect.TypeOf((*uint64)(nil)).Elem()) {
			return "", errors.New("values in the psid option field must be numbers")
		}
	}
	psid := reflect.ValueOf(values[0]).Convert(reflect.TypeOf((*uint64)(nil)).Elem())
	if psid.Uint() > math.MaxUint16 {
		return "", errors.Errorf("psid value must not be greater than %d", math.MaxUint16)
	}
	psidLength := reflect.ValueOf(values[1]).Convert(reflect.TypeOf((*uint64)(nil)).Elem())
	if psidLength.Uint() > math.MaxUint8 {
		return "", errors.Errorf("psid length value must not be greater than %d", math.MaxUint8)
	}
	if textFormat {
		return fmt.Sprintf("%d/%d", values[0], values[1]), nil
	}
	return fmt.Sprintf("%04X%02X", psid.Uint(), psidLength.Uint()), nil
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
		return "", errors.New("multiple values in the FQDN option field")
	}
	value, ok := values[0].(string)
	if !ok {
		return "", errors.New("FQDN option field is not a string value")
	}
	fqdn := storkutil.ParseFqdn(value)
	if fqdn == nil {
		return "", errors.New("FQDN option field contains an invalid value")
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
