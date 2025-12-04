package storkutil

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// Normalizes JSON. One of the cases for using this function is to
// normalize Kea JSON configuration. Kea accepts JSON files that don't
// strictly follow the standard.
// Specifically, it allows:
//   - trailing commas in arrays and objects
//   - C-style comments (both single line // and multi-line /* */)
//   - Python-style comments (# ...)
//
// This function removes these non-standard constructs so that the resulting
// JSON can be parsed by standard JSON parsers.
//
// The output JSON trims all whitespace except for Unix line breaks.
//
// Inspired by https://github.com/muhammadmuzzammil1998/jsonc.
func NormalizeJSON(input []byte) []byte {
	// This function operates on the UTF-8 characters (runes). This object allows
	// to write them to a byte buffer efficiently and handy.
	var output bytes.Buffer
	// Reader for proper handling of UTF-8 characters (runes).
	inputReader := bytes.NewReader(input)

	// True if the parser is currently inside a single-line comment
	// (Python-style # or C-style //).
	isSingleLineComment := false
	// True if the parser is currently inside a C-style multi-line comment
	// (/* ... */).
	isMultiLineComment := false
	// True if the parser is currently inside a string.
	isString := false
	// True if the previous character was a slash, indicating potential
	// opening of a C-style comment.
	remainingSlash := false
	// True if the previous non-whitespace character was a comma, indicating
	// potential trailing comma.
	remainingComma := false

	// Current character being processed.
	currentChar := rune(0)
	var err error

	for {
		// Read the next character and keep track of the previous character.
		// It reads the bytes by rune to properly handle UTF-8 encoded JSONs.
		previousChar := currentChar
		currentChar, _, err = inputReader.ReadRune()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				// The only possible error here is EOF, so it should not happen.
				logrus.WithError(err).Error("Failed to read rune from input while normalizing JSON")
			}
			break
		}

		// Exiting from states in which most of the normalization rules
		// are ignored.
		switch {
		case isSingleLineComment:
			// Disable the single-line comment mode at the end of the line.
			if currentChar == '\n' {
				isSingleLineComment = false
			}
			continue
		case isMultiLineComment:
			// Disable the block comment mode at the closing tag.
			if previousChar == '*' && currentChar == '/' {
				isMultiLineComment = false
			}
			continue
		case isString:
			// Disable the string mode at the ending quote, unless it is escaped.
			if currentChar == '"' && previousChar != '\\' {
				isString = false
			}
			output.WriteRune(currentChar)
			continue
		}

		// Entering into C++-style comment states.
		if remainingSlash {
			remainingSlash = false
			switch currentChar {
			case '/':
				// Single-line comment.
				isSingleLineComment = true
				continue
			case '*':
				// Multi line comment.
				isMultiLineComment = true
				continue
			}
			// It was not a comment, write the slash we have seen before.
			output.WriteRune('/')
		}

		// Detecting potential comment openings.
		switch currentChar {
		case '/':
			// Potential C-style comment.
			remainingSlash = true
			continue
		case '#':
			// Python-style single line comment.
			isSingleLineComment = true
			continue
		}

		// Trim whitespaces outside of strings.
		// The whitespaces outside of strings are not significant in JSON.
		// However, it must be done after handling comments because C-style
		// opening characters must not be separated by whitespace.
		if unicode.IsSpace(currentChar) {
			continue
		}
		// Check for trailing commas in objects and arrays.
		// They are not standard-compliant but Kea allows them.
		if remainingComma {
			remainingComma = false
			if currentChar != '}' && currentChar != ']' {
				// It wasn't a trailing comma, write it to the output.
				output.WriteRune(',')
			}
		}

		// Handle other special characters.
		switch currentChar {
		case ',':
			// Potential trailing comma.
			remainingComma = true
			continue
		case '"':
			// Entering into string mode.
			isString = true
			output.WriteRune(currentChar)
			continue
		}

		// Normal character, just write it to the output.
		output.WriteRune(currentChar)
	}

	return output.Bytes()
}
