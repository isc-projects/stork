package keaconfig

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// A structure combining a structure with known (supported) configuration
// parameters and a map of unknown (unsupported) configuration parameters.
// The typical use case is to ensure that unknown parameters are not lost
// when unmarshalling and marshalling the configuration. To distinguish
// unknown parameters it is expected that the structure specified as a
// generic type is annotated with the json tag holding the parameter name.
type WithUnknown[T any] struct {
	Known   T
	Unknown map[string]any `json:"-"`
}

// Collects the names of the parameters associated with the structure
// fields via the json tag. The argument is a reflected type of the
// structure. If it is not a structure type, an empty slice is returned.
func collectTags(t reflect.Type) (tags []string) {
	if t.Kind() != reflect.Struct {
		return
	}
	// Go over the fields of the structure and identify the json tags.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			// Anonymous field is embedded in the structure. We're not
			// interested in its tags. We want to extract the tags
			// associated with its fields. Therefore, we call the
			// collectTags function recursively for the embedded
			// structure type.
			tags = append(tags, collectTags(field.Type)...)
		}
		// Get the json tag for the field.
		jsonTag := strings.Split(field.Tag.Get("json"), ",")
		if len(jsonTag) == 0 || jsonTag[0] == "-" || jsonTag[0] == "" {
			// If the tag does not exist, specifies empty parameter name or
			// is set to "-" we skip the field.
			continue
		}
		// Add the parameter name to the list of tags.
		tags = append(tags, jsonTag[0])
	}
	return tags
}

// Parse the structure specified as a generic type together with unknown
// parameters.
func (w *WithUnknown[T]) UnmarshalJSON(data []byte) error {
	// Start by parsing the known fields.
	if err := json.Unmarshal(data, &w.Known); err != nil {
		return errors.Wrap(err, "problem unmarshalling known configuration parameters")
	}

	// If the type is not a structure, there are no unknown parameters.
	// We can return early.
	knownType := reflect.TypeOf((*T)(nil)).Elem()
	if knownType.Kind() != reflect.Struct {
		return nil
	}

	// Collect the tags into the map.
	tags := make(map[string]bool)
	for _, tag := range collectTags(reflect.TypeOf((*T)(nil)).Elem()) {
		tags[tag] = true
	}

	// To identify unknown parameters we need to unmarshal the entire configuration
	// into a map.
	var all map[string]any
	if err := json.Unmarshal(data, &all); err != nil {
		return errors.Wrap(err, "problem unmarshalling entire configuration into a map")
	}

	// Walk over the map and collect unknown parameters.
	for k, v := range all {
		if _, exists := tags[k]; !exists {
			// If the parameter is not in the list of known parameters,
			// it is unknown.
			if w.Unknown == nil {
				w.Unknown = make(map[string]any)
			}
			w.Unknown[k] = v
		}
	}
	return nil
}

// Serialize the structure specified as a generic type together with unknown
// parameters.
func (w WithUnknown[T]) MarshalJSON() ([]byte, error) {
	type alias *T
	// Serialize the known parameters.
	kb, err := json.Marshal((alias)(&w.Known))
	if err != nil {
		return nil, errors.Wrap(err, "problem marshalling known configuration parameters")
	}

	// If the type is not a structure, there are no unknown parameters.
	// We can return early.
	knownType := reflect.TypeOf((*T)(nil)).Elem()
	if knownType.Kind() != reflect.Struct {
		return kb, nil
	}

	// Convert known parameters to a map.
	var m map[string]any
	if err := json.Unmarshal(kb, &m); err != nil {
		return nil, errors.Wrap(err, "problem unmarshalling known configuration parameters into a map")
	}

	// Identify known parameter names. It will be used to distinguish unknown
	// parameters from known parameters.
	tags := make(map[string]bool)
	for _, tag := range collectTags(reflect.TypeOf((*T)(nil)).Elem()) {
		tags[tag] = true
	}

	// Append unknown parameters to the known parameters.
	for k, v := range w.Unknown {
		if _, exists := tags[k]; !exists {
			m[k] = v
		}
	}

	// Serialize the entire configuration.
	marshalled, err := json.Marshal(m)
	return marshalled, errors.Wrap(err, "problem marshalling known and unknown configuration parameters")
}
