package keaconfig

import (
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// Decodes a map into structure with ignoring hyphens. Hyphens are
// used in the Kea configurations but the mapstructure does not
// take them into account and fails to match the keys comprising
// the hyphens with the struct fields. It also sets the squash
// flag to true to treat embedded structures as if their fields
// belong to the parent structure.
func decode(input interface{}, output interface{}) error {
	decoderConfig := mapstructure.DecoderConfig{
		// Create a custom matcher that removes hyphens.
		MatchName: func(mapKey, fieldName string) bool {
			return strings.EqualFold(strings.ReplaceAll(mapKey, "-", ""), fieldName)
		},
		Squash: true,
		Result: output,
	}
	decoder, _ := mapstructure.NewDecoder(&decoderConfig)
	err := errors.WithStack(decoder.Decode(input))
	return err
}
