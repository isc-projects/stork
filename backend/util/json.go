package storkutil

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Converts specified interface to int64. It expects that the interface already
// has int64 type or json.Number type convertible to int64.
func ConvertJSONInt64(value interface{}) (valueInt64 int64, err error) {
	var ok bool
	if valueInt64, ok = value.(int64); ok {
		return
	}
	if valueJSON, ok := value.(json.Number); ok {
		if valueInt64, err = valueJSON.Int64(); err == nil {
			return
		}
	}
	err = errors.Errorf("value %v is not an int64 number", value)
	return
}

// It extracts specified value from the map of interfaces and converts it to
// int64. It expects that the value in the map is already an int64 value or
// a json.Number convertible to int64.
func ExtractJSONInt64(container map[string]interface{}, key string) (int64, error) {
	if value, ok := container[key]; ok {
		return ConvertJSONInt64(value)
	}
	return 0, errors.Errorf("value not found in the container for key %s", key)
}
