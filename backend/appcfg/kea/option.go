package keaconfig

import (
	"fmt"
	"strings"

	errors "github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Represents a DHCP option in the format used by Kea (i.e., an item of the
// option-data list).
type SingleOptionData struct {
	AlwaysSend bool   `json:"always-send,omitempty"`
	Code       uint16 `json:"code,omitempty"`
	CSVFormat  bool   `json:"csv-format"`
	Data       string `json:"data,omitempty"`
	Name       string `json:"name,omitempty"`
	Space      string `json:"space,omitempty"`
}

// Creates a SingleOptionData instance from the DHCP option model used
// by Stork (e.g., from an option held in the Stork database). If the
// option has a definition, it uses the Kea's csv-format setting and
// specifies the option value as a comma separated list. Otherwise, it
// converts the option fields to a hex form and sets the csv-format to
// false. The lookup interface must not be nil.
func CreateSingleOptionData(daemonID int64, lookup DHCPOptionDefinitionLookup, option dhcpmodel.DHCPOptionAccessor) (*SingleOptionData, error) {
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
		case dhcpmodel.BinaryField:
			value, err = ConvertBinaryField(field)
		case dhcpmodel.StringField:
			value, err = ConvertStringField(field, data.CSVFormat)
		case dhcpmodel.BoolField:
			value, err = ConvertBoolField(field, data.CSVFormat)
		case dhcpmodel.Uint8Field, dhcpmodel.Uint16Field, dhcpmodel.Uint32Field, dhcpmodel.Int8Field, dhcpmodel.Int16Field, dhcpmodel.Int32Field:
			value, err = ConvertIntField(field, data.CSVFormat)
		case dhcpmodel.IPv4AddressField:
			value, err = ConvertIPv4AddressField(field, data.CSVFormat)
		case dhcpmodel.IPv6AddressField:
			value, err = ConvertIPv6AddressField(field, data.CSVFormat)
		case dhcpmodel.IPv6PrefixField:
			value, err = ConvertIPv6PrefixField(field, data.CSVFormat)
		case dhcpmodel.PsidField:
			value, err = ConvertPsidField(field, data.CSVFormat)
		case dhcpmodel.FqdnField:
			value, err = ConvertFqdnField(field, data.CSVFormat)
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

// Represents a DHCP option and implements DHCPOption interface. It is returned
// by the CreateDHCPOption function.
type DHCPOption struct {
	AlwaysSend  bool
	Code        uint16
	Encapsulate string
	Fields      []dhcpmodel.DHCPOptionFieldAccessor
	Name        string
	Space       string
	Universe    storkutil.IPType
}

// Creates an instance of a DHCP option in Stork from the option representation
// in Kea.
func CreateDHCPOption(optionData SingleOptionData, universe storkutil.IPType, lookup DHCPOptionDefinitionLookup) (dhcpmodel.DHCPOptionAccessor, error) {
	option := DHCPOption{
		AlwaysSend: optionData.AlwaysSend,
		Code:       optionData.Code,
		Name:       optionData.Name,
		Space:      optionData.Space,
		Universe:   universe,
	}
	data := strings.TrimSpace(optionData.Data)

	// Option encapsulation.
	def := lookup.Find(0, option)
	if def != nil {
		// If the option definition is known, let's take the encapsulated option
		// space name from it.
		option.Encapsulate = def.GetEncapsulate()
	} else {
		// Generate the encapsulated option space name because option
		// definition does not exist in Stork for this option.
		switch option.Space {
		case dhcpmodel.DHCPv4OptionSpace, dhcpmodel.DHCPv6OptionSpace:
			option.Encapsulate = fmt.Sprintf("option-%d", option.Code)
		default:
			option.Encapsulate = fmt.Sprintf("%s.%d", option.Space, option.Code)
		}
	}

	// There is nothing to do if the option is empty.
	if len(data) == 0 || (def != nil && def.GetType() == EmptyOption) {
		return option, nil
	}

	// Option data specified as comma separated values.
	if optionData.CSVFormat {
		values := strings.Split(data, ",")
		for i, raw := range values {
			v := strings.TrimSpace(raw)
			var field dhcpmodel.DHCPOptionFieldAccessor
			if def != nil {
				fieldType, ok := GetDHCPOptionDefinitionFieldType(def, i)
				if !ok {
					break
				}
				// We know option definition so we expect that our option
				// adheres to the specific format. Try to parse the option
				// field with checking whether or not its value has that
				// format.
				var err error
				if field, err = ParseDHCPOptionField(fieldType, v); err != nil {
					return nil, err
				}
			} else {
				// We don't know the option definition so we will need to
				// try to infer option field data format.
				field = inferDHCPOptionField(v)
			}
			option.Fields = append(option.Fields, field)
		}
		return option, nil
	}

	// If the csv-format is false the option payload is specified using a string
	// of hexadecimal digits. Sanitize colons and whitespaces.
	data = strings.ReplaceAll(strings.ReplaceAll(data, " ", ""), ":", "")
	field := dhcpOptionField{
		FieldType: dhcpmodel.BinaryField,
		Values:    []any{data},
	}
	option.Fields = append(option.Fields, field)

	return option, nil
}
