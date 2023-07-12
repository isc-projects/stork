package keaconfig

import (
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	errors "github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Represents a DHCP option field and implements DHCPOptionField interface. It
// is returned by the CreateDHCPOption function.
type dhcpOptionField struct {
	FieldType string
	Values    []any
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
func (option DHCPOption) GetFields() (returnedFields []dhcpmodel.DHCPOptionFieldAccessor) {
	return option.Fields
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

// Tries to infer option field type from its value and converts it to the
// format used in Stork. The resulting format comprises an option field type
// and the corresponding field value(s). It has some limitations:
//
//   - It is unable to recognize an exact integer type, therefore it returns
//     all positive numbers as uint32 fields and all negative numbers as
//     int32 fields.
//   - It is unable to differentiate between partial FQDN and a regular string,
//     therefore it returns string field for partial FQDNs.
//
// The function is used temporarily until we add support for Kea option
// definitions. However, even then, this function may be useful in cases when
// the definition is not available for any reason.
func inferDHCPOptionField(value string) dhcpOptionField {
	var field dhcpOptionField

	// Is it a bool value?
	if bv, err := ParseBoolField(value); err == nil {
		field = dhcpOptionField{
			FieldType: dhcpmodel.BoolField,
			Values:    []any{bv},
		}
		return field
	}
	// Is it an unsigned number?
	if iv, err := ParseUint32Field(value); err == nil {
		field = dhcpOptionField{
			FieldType: dhcpmodel.Uint32Field,
			Values:    []any{iv},
		}
		return field
	}
	// Is it a negative number.
	if iv, err := ParseInt32Field(value); err == nil {
		field = dhcpOptionField{
			FieldType: dhcpmodel.Int32Field,
			Values:    []any{iv},
		}
		return field
	}
	// Is it an IP address or prefix?
	if ip, err := ParseIPField(value); err == nil {
		switch ip.Protocol {
		// Is it an IPv4 address?
		case storkutil.IPv4:
			field = dhcpOptionField{
				FieldType: dhcpmodel.IPv4AddressField,
				Values:    []any{value},
			}
			return field
		// Is it an IPv6 address or prefix?
		case storkutil.IPv6:
			// Is it a prefix?
			if ip.Prefix {
				field = dhcpOptionField{
					FieldType: dhcpmodel.IPv6PrefixField,
					Values: []any{
						ip.NetworkPrefix,
						ip.PrefixLength,
					},
				}
			} else {
				// Is it an IPv6 address?
				field = dhcpOptionField{
					FieldType: dhcpmodel.IPv6AddressField,
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
				FieldType: dhcpmodel.FqdnField,
				Values:    []any{value},
			}
			return field
		}
	}
	// Is it PSID?
	if psid, psidLen, err := ParsePsidField(value); err == nil {
		field = dhcpOptionField{
			FieldType: dhcpmodel.PsidField,
			Values:    []any{psid, psidLen},
		}
		return field
	}
	// It must be a string.
	field = dhcpOptionField{
		FieldType: dhcpmodel.StringField,
		Values:    []any{value},
	}
	return field
}

// Parses the DHCP option field value specified as a string given
// its type specified as fieldType. If the value doesn't match the
// specified type an error is returned.
func ParseDHCPOptionField(fieldType dhcpmodel.DHCPOptionFieldType, value string) (dhcpmodel.DHCPOptionFieldAccessor, error) {
	field := dhcpOptionField{
		FieldType: fieldType,
	}
	switch fieldType {
	case dhcpmodel.BoolField:
		bv, err := ParseBoolField(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{bv}
		return field, nil
	case dhcpmodel.Uint8Field:
		iv, err := ParseUint8Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.Uint16Field:
		iv, err := ParseUint16Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.Uint32Field:
		iv, err := ParseUint32Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.Int8Field:
		iv, err := ParseInt8Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.Int16Field:
		iv, err := ParseInt16Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.Int32Field:
		iv, err := ParseInt32Field(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{iv}
	case dhcpmodel.IPv4AddressField:
		ip, err := ParseIPField(value)
		if err != nil {
			return nil, err
		}
		if ip.Protocol != storkutil.IPv4 {
			return nil, errors.Errorf("%s is not a valid IPv4 address option field value", value)
		}
		field.Values = []any{value}
	case dhcpmodel.IPv6AddressField:
		ip, err := ParseIPField(value)
		if err != nil {
			return nil, err
		}
		if ip.Protocol != storkutil.IPv6 || ip.Prefix {
			return nil, errors.Errorf("%s is not a valid IPv6 address option field value", value)
		}
		field.Values = []any{value}
	case dhcpmodel.IPv6PrefixField:
		ip, err := ParseIPField(value)
		if err != nil {
			return nil, err
		}
		if !ip.Prefix {
			return nil, errors.Errorf("%s is not a valid IPv6 prefix option field", value)
		}
		field.Values = []any{
			ip.NetworkPrefix,
			ip.PrefixLength,
		}
	case dhcpmodel.FqdnField:
		_, err := storkutil.ParseFqdn(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{value}
	case dhcpmodel.PsidField:
		psid, psidLen, err := ParsePsidField(value)
		if err != nil {
			return nil, err
		}
		field.Values = []any{psid, psidLen}
	default:
		field.Values = []any{value}
	}
	return field, nil
}

// Parse boolean option field.
func ParseBoolField(value string) (bool, error) {
	bv, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.Errorf("%s is not a valid bool option field value", value)
	}
	return bv, nil
}

// Parse uint8 option field.
func ParseUint8Field(value string) (uint8, error) {
	iv, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint8 option field value", value)
	}
	return uint8(iv), nil
}

// Parse uint16 option field.
func ParseUint16Field(value string) (uint16, error) {
	iv, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint16 option field value", value)
	}
	return uint16(iv), nil
}

// Parse uint32 option field.
func ParseUint32Field(value string) (uint32, error) {
	iv, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint32 option field value", value)
	}
	return uint32(iv), nil
}

// Parse int8 option field.
func ParseInt8Field(value string) (int8, error) {
	iv, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint8 option field value", value)
	}
	return int8(iv), nil
}

// Parse int16 option field.
func ParseInt16Field(value string) (int16, error) {
	iv, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint16 option field value", value)
	}
	return int16(iv), nil
}

// Parse int32 option field.
func ParseInt32Field(value string) (int32, error) {
	iv, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, errors.Errorf("%s is not a valid uint32 option field value", value)
	}
	return int32(iv), nil
}

// Parse IPv4 address, IPv6 address or IPv6 prefix option field.
func ParseIPField(value string) (*storkutil.ParsedIP, error) {
	ip := storkutil.ParseIP(value)
	if ip == nil {
		return nil, errors.Errorf("%s is neither an IP address nor prefix", value)
	}
	return ip, nil
}

// Parse PSID option field.
func ParsePsidField(value string) (uint16, uint8, error) {
	pv := strings.Split(value, "/")
	if len(pv) != 2 {
		return 0, 0, errors.Errorf("psid value %s should have the psid/psidLength format", value)
	}
	psid, err := strconv.ParseUint(pv[0], 10, 16)
	if err != nil {
		return 0, 0, errors.Errorf("psid value %s must be an uint16 integer", pv[0])
	}
	psidLen, err := strconv.ParseUint(pv[1], 10, 8)
	if err != nil {
		return 0, 0, errors.Errorf("psid length value %s must be an uint8 integer", pv[1])
	}
	return uint16(psid), uint8(psidLen), nil
}

// Converts a binary option field from Stork to Kea format. The conversion
// is the same regardless of the csv-format setting. It removes the colons
// and spaces if they are used as separators.
func ConvertBinaryField(field dhcpmodel.DHCPOptionFieldAccessor) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in binary option field, have %d", len(values))
	}
	output, ok := values[0].(string)
	if !ok {
		return "", errors.New("binary option field is not a string value")
	}
	if len(output) == 0 {
		return "", errors.New("binary option field value must not be empty")
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
func ConvertStringField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
func ConvertBoolField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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

// Converts an uint8, uint16, uint32, int8, int16 or int32 option field from
// Stork to Kea format. It expects that there is one option field value and
// that this value is a number. If the textFormat is set to true, it returns
// the value converted to a string. Otherwise, it converts the value to the
// hex format.
func ConvertIntField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
	values := field.GetValues()
	if len(values) != 1 {
		return "", errors.Errorf("require one value in int option field, have %d", len(values))
	}
	if !storkutil.IsWholeNumber(values[0]) {
		return "", errors.New("int option field value is not a valid number")
	}
	value := reflect.ValueOf(values[0])
	ivalue := value.Convert(reflect.TypeOf((*int64)(nil)).Elem())
	switch field.GetFieldType() {
	case dhcpmodel.Uint8Field:
		if ivalue.Int() < 0 || ivalue.Int() > math.MaxUint8 {
			return "", errors.Errorf("uint8 option field value must be between 0 and %d", math.MaxUint8)
		}
		if !textFormat {
			return fmt.Sprintf("%02X", ivalue.Int()), nil
		}
	case dhcpmodel.Uint16Field:
		if ivalue.Int() < 0 || ivalue.Int() > math.MaxUint16 {
			return "", errors.Errorf("uint16 option field value must be between 0 and %d", math.MaxUint16)
		}
		if !textFormat {
			return fmt.Sprintf("%04X", ivalue.Int()), nil
		}
	case dhcpmodel.Uint32Field:
		if ivalue.Int() < 0 || ivalue.Int() > math.MaxUint32 {
			return "", errors.Errorf("uint32 option field value must be between 0 and %d", uint32(math.MaxUint32))
		}
		if !textFormat {
			return fmt.Sprintf("%08X", ivalue.Int()), nil
		}
	case dhcpmodel.Int8Field:
		if ivalue.Int() < math.MinInt8 || ivalue.Int() > math.MaxInt8 {
			return "", errors.Errorf("int8 option field value must be between %d and %d", math.MinInt8, math.MaxInt8)
		}
		if !textFormat {
			// Cast integer to unsigned integer before hex conversion.
			return fmt.Sprintf("%02X", uint8(ivalue.Int())), nil
		}
	case dhcpmodel.Int16Field:
		if ivalue.Int() < math.MinInt16 || ivalue.Int() > math.MaxInt16 {
			return "", errors.Errorf("int16 option field value must be between %d and %d", math.MinInt16, math.MaxInt16)
		}
		if !textFormat {
			// Cast integer to unsigned integer before hex conversion.
			return fmt.Sprintf("%04X", uint16(ivalue.Int())), nil
		}
	case dhcpmodel.Int32Field:
		if ivalue.Int() < math.MinInt32 || ivalue.Int() > math.MaxInt32 {
			return "", errors.Errorf("int32 option field value must be between %d and %d", math.MinInt32, math.MaxInt32)
		}
		if !textFormat {
			// Cast integer to unsigned integer before hex conversion.
			return fmt.Sprintf("%08X", uint32(ivalue.Int())), nil
		}
	}
	return fmt.Sprintf("%d", values[0]), nil
}

// Converts an IPv4 address option field from Stork to Kea format. It expects
// that there is one option field value and that this value is a string with
// a valid IPv4 address. If the textFormat is set to true, it returns the
// original string. Otherwise, it converts the IPv4 address to the hex format.
func ConvertIPv4AddressField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
func ConvertIPv6AddressField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
func ConvertIPv6PrefixField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
		return storkutil.FormatCIDRNotation(ip, int(prefixLen.Int())), nil
	}
	return fmt.Sprintf("%s%02X", storkutil.BytesToHex(parsed.IP), prefixLen.Int()), nil
}

// Converts a PSID option field from Stork to Kea format. It expects that
// there are two option field values, one is a uint16 value specifying PSID,
// and the second one is a uint8 value specifying PSID length. If the
// textFormat is set to true, it returns the option field value in the
// PSID/PSIDLen format. Otherwise, it converts the option fields to the hex
// format and returns them concatenated.
func ConvertPsidField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
func ConvertFqdnField(field dhcpmodel.DHCPOptionFieldAccessor, textFormat bool) (string, error) {
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
