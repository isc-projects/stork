package storkutil

import (
	"encoding/json"

	"github.com/pkg/errors"
)

const nullLiteral = "null"

// A wrapper around marshalled values making it possible to represent
// explicit null values in JSON. The native golang marshaller does not
// provide a way to differentiate between an explicit null value and
// unspecified value. Some of our use cases in Stork require explicit
// null values. Specifically, when we merge two Kea configurations, an
// explicit null value indicates that it should be deleted from the
// target configuration. It is different than not including this value
// in the source configuration. If the value is not included, the corresponding
// value in the target configuration is left untouched.
// The Nullable type is marshalled as the inner value type if the value
// is non-nil. If the specified value is nil, the marshaller outputs
// the JSON null value.
type Nullable[T any] struct {
	value *T
}

// A wrapper around marshalled arrays making it possible to represent
// explicit null values in JSON. It is similar to Nullable type but it
// wraps slices instead of a single value.
type NullableArray[T any] struct {
	value []T
}

// Instantiates a new nullable value from its pointer.
func NewNullable[T any](value *T) *Nullable[T] {
	return &Nullable[T]{
		value: value,
	}
}

// Instantiates a new nullable value.
func NewNullableFromValue[T any](value T) *Nullable[T] {
	return &Nullable[T]{
		value: &value,
	}
}

// Returns wrapped value.
func (v Nullable[T]) GetValue() *T {
	return v.value
}

// Marshals the Nullable value. It returns the serialized wrapped value
// it is non-nil. Otherwise, it returns null JSON value.
func (v Nullable[T]) MarshalJSON() ([]byte, error) {
	if v.value != nil {
		marshalled, err := json.Marshal(*v.value)
		return marshalled, errors.Wrapf(err, "failed to marshal nullable value %v", *v.value)
	}
	return []byte(nullLiteral), nil
}

// Parses JSON value into Nullable.
func (v *Nullable[T]) UnmarshalJSON(serial []byte) error {
	if string(serial) != nullLiteral {
		var decoded T
		if err := json.Unmarshal(serial, &decoded); err != nil {
			return errors.Wrapf(err, "failed to unmarshal into nullable value: %s", string(serial))
		}
		v.value = &decoded
	} else {
		v.value = nil
	}
	return nil
}

// Marshals the NullableArray value. It returns the serialized wrapped value
// it is non-nil. Otherwise, it returns null JSON value.
func NewNullableArray[T any](value []T) *NullableArray[T] {
	return &NullableArray[T]{
		value: value,
	}
}

// Returns wrapped array value.
func (v NullableArray[T]) GetValue() []T {
	return v.value
}

// Marshals the Nullable value. It returns the serialized wrapped value
// it is non-nil. Otherwise, it returns null JSON value.
func (v NullableArray[T]) MarshalJSON() ([]byte, error) {
	if v.value != nil {
		marshalled, err := json.Marshal(v.value)
		return marshalled, errors.Wrapf(err, "failed to marshal nullable array: %v", v.value)
	}
	return []byte(nullLiteral), nil
}

// Parses JSON value into NullableArray.
func (v *NullableArray[T]) UnmarshalJSON(serial []byte) error {
	if string(serial) != nullLiteral {
		var decoded []T
		if err := json.Unmarshal(serial, &decoded); err != nil {
			return errors.Wrapf(err, "failed to unmarshal into nullable array: %s", string(serial))
		}
		v.value = decoded
	} else {
		v.value = nil
	}
	return nil
}

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
