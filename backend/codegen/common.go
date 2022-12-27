package codegen

import (
	"strings"

	"github.com/pkg/errors"
)

// Common function used by the code generator and the engines to parse command
// line arguments specified using the <key>:<value> notation.
func parseMappings(mappings []string, output map[string]string) error {
	for _, m := range mappings {
		split := strings.Split(m, ":")
		if len(split) != 2 {
			return errors.Errorf("invalid mapping %s; expected <key>:<value>", m)
		}
		output[split[0]] = split[1]
	}
	return nil
}
