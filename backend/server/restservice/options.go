package restservice

import (
	"fmt"
	"strconv"

	errors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// Converts DHCP options from the REST API format to the database format. The
// received options have hierarchical structure (i.e., suboptions are included
// in the options). This function flattens this structure by placing suboptions
// at the same level as the options. The linkage between the two is maintained
// using the "encapsulate" and "space" fields. If an option encapsulates an
// option space which is assigned to a suboption, this suboption belongs to
// the option. This is the same concept as in Kea. Option field values are
// converted to suitable types. For example, if an option field has a uint8
// value, the received option field value (REST API value) string is converted
// to the uint8 type in the database model. This function should be called
// with the recursionLevel value of 0. It supports up to three recursion
// levels, i.e., top-level option with suboptions with suboptions. If there
// is an option at deeper level, it is excluded from the result.
func flattenDHCPOptions(optionSpace string, restOptions []*models.DHCPOption, recursionLevel int) ([]dbmodel.DHCPOption, error) {
	var options []dbmodel.DHCPOption
	// Break if recursion level exceeded.
	if recursionLevel >= 3 {
		return options, nil
	}
	// Convert each option.
	for _, restOption := range restOptions {
		option := dbmodel.DHCPOption{
			AlwaysSend:  restOption.AlwaysSend,
			Code:        uint16(restOption.Code),
			Encapsulate: restOption.Encapsulate,
			Universe:    storkutil.IPType(restOption.Universe),
		}
		// The option space should be set for suboptions.
		if len(optionSpace) > 0 {
			option.Space = optionSpace
		} else {
			// Set top-level option space.
			if storkutil.IPType(restOption.Universe) == storkutil.IPv4 {
				option.Space = keaconfig.DHCPv4OptionSpace
			} else {
				option.Space = keaconfig.DHCPv6OptionSpace
			}
		}
		// Go over the option fields belonging to our options.
		for _, restField := range restOption.Fields {
			field := dbmodel.DHCPOptionField{
				FieldType: restField.FieldType,
			}
			// An option field must always have at least one value.
			if len(restField.Values) == 0 {
				return nil, errors.New("no values in the option field")
			}
			// Validate and convert specific option fields.
			switch field.FieldType {
			case keaconfig.Uint8Field:
				uintValue, err := strconv.ParseUint(restField.Values[0], 10, 8)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert option field value %s to uint8", restField.Values[0])
				}
				field.Values = append(field.Values, uint8(uintValue))
			case keaconfig.Uint16Field:
				uintValue, err := strconv.ParseUint(restField.Values[0], 10, 16)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert option field value %s to uint16", restField.Values[0])
				}
				field.Values = append(field.Values, uint16(uintValue))
			case keaconfig.Uint32Field:
				uintValue, err := strconv.ParseUint(restField.Values[0], 10, 32)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert option field value %s to uint32", restField.Values[0])
				}
				field.Values = append(field.Values, uint32(uintValue))
			case keaconfig.BoolField:
				boolValue, err := strconv.ParseBool(restField.Values[0])
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert option field value %s to boolean", restField.Values[0])
				}
				field.Values = append(field.Values, boolValue)
			case keaconfig.IPv6PrefixField:
				if len(restField.Values) < 2 {
					return nil, errors.New("invalid number of values in the IPv6 prefix option field")
				}
				field.Values = append(field.Values, restField.Values[0])
				prefixLen, err := strconv.ParseUint(restField.Values[1], 10, 8)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert IPv6 prefix length %s to a number", restField.Values[1])
				}
				field.Values = append(field.Values, uint8(prefixLen))
			case keaconfig.PsidField:
				if len(restField.Values) < 2 {
					return nil, errors.New("invalid number of values in the PSID option field")
				}
				psid, err := strconv.ParseUint(restField.Values[0], 10, 16)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert PSID %s to a number", restField.Values[0])
				}
				psidLen, err := strconv.ParseUint(restField.Values[1], 10, 8)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert PSID length %s to a number", restField.Values[1])
				}
				field.Values = append(field.Values, uint16(psid), uint8(psidLen))
			default:
				field.Values = append(field.Values, restField.Values[0])
			}
			option.Fields = append(option.Fields, field)
		}
		if len(restOption.Options) > 0 {
			// Convert suboptions recursively.
			suboptions, err := flattenDHCPOptions(option.Encapsulate, restOption.Options, recursionLevel+1)
			if err != nil {
				return nil, err
			}
			options = append(options, suboptions...)
		}
		options = append(options, option)
	}
	return options, nil
}

// Converts DHCP options from the database model to REST API format. The options
// stored in the database have flat structure. Suboptions are associated with the
// parent options via option spaces. This function uses option spaces to put the
// options into a hierarchical structure used in the REST API. It processes the
// options recursively with a three-level limit (i.e., top level options with
// suboptions with suboptions). All option field values are converted to strings.
func unflattenDHCPOptions(options []dbmodel.DHCPOption, space string, recursionLevel int) []*models.DHCPOption {
	var restOptions []*models.DHCPOption
	// Break if recursion level exceeded.
	if recursionLevel >= 3 {
		return restOptions
	}
	for _, option := range options {
		// If it is a top-level option the option space argument is empty.
		// In that case, select options belonging to dhcp4 or dhcp6 option
		// spaces. Otherwise, check if the specified option space matches
		// the current option's space. If so, convert the option.
		if (space == "" && (option.Space == keaconfig.DHCPv4OptionSpace || option.Space == keaconfig.DHCPv6OptionSpace)) ||
			space == option.Space {
			restOption := &models.DHCPOption{
				AlwaysSend:  option.AlwaysSend,
				Code:        int64(option.Code),
				Encapsulate: option.Encapsulate,
				Universe:    int64(option.Universe),
			}
			for _, field := range option.Fields {
				restField := &models.DHCPOptionField{
					FieldType: field.FieldType,
				}
				// Convert option values to strings.
				for _, v := range field.Values {
					restField.Values = append(restField.Values, fmt.Sprintf("%v", v))
				}
				restOption.Fields = append(restOption.Fields, restField)
			}
			// Append suboptions recursively for the encapsulated option space.
			if len(restOption.Encapsulate) > 0 {
				restOption.Options = unflattenDHCPOptions(options, restOption.Encapsulate, recursionLevel+1)
			}
			restOptions = append(restOptions, restOption)
		}
	}
	return restOptions
}
