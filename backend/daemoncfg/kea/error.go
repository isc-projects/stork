package keaconfig

import (
	"fmt"
)

// An error returned on attempt to set a parameter that doesn't belong to
// the current configuration.
type UnsupportedConfigParameter struct {
	parameterName string
}

// Create new instance of the UnsupportedConfigParameter.
func NewUnsupportedConfigParameter(parameterName string) error {
	return &UnsupportedConfigParameter{
		parameterName: parameterName,
	}
}

// Returns error string.
func (e UnsupportedConfigParameter) Error() string {
	return fmt.Sprintf("unsupported configuration parameter %s", e.parameterName)
}
