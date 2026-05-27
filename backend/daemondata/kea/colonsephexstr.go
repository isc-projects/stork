package keadata

import (
	"encoding/json"
	"strings"

	"github.com/go-pg/pg/v10/types"
)

// Custom support for colon-separated hexadecimal-encoded byte strings in Go-PG.
// This stores DUIDs or Client IDs in the canonical `01:02:03` format, but inserts
// them into the database in a format supported by `bytea` columns.
type ColonSepHexStr struct {
	String string
}

// newColonSeparatedHexStr wraps an existing string into a [colonSeparatedHexStr]
// without performing any validation.
func NewColonSepHexStr(val *string) *ColonSepHexStr {
	if val == nil {
		return nil
	}
	return &ColonSepHexStr{String: *val}
}

func NewColonSepHexStrZero() *ColonSepHexStr {
	empty := ""
	return NewColonSepHexStr(&empty)
}

// ToString returns the string inside the [ColonSepHexStr], or the empty string
// if provided with a nil receiver.
func (s *ColonSepHexStr) ToString() string {
	if s == nil {
		return ""
	}
	return s.String
}

// Define a variable of the structure type so that the compiler warns about
// noncompliance with the *serializer* interface.
var _ types.ValueAppender = (*ColonSepHexStr)(nil)

// AppendValue writes the value from the receiver into a byte array in the
// '\x00112233' format expected by PostgreSQL.
func (s *ColonSepHexStr) AppendValue(b []byte, quote int) ([]byte, error) {
	if quote == 1 {
		b = append(b, '\'')
	}
	b = append(b, []byte("\\x")...)
	noColons := strings.ReplaceAll(s.String, ":", "")
	b = append(b, []byte(noColons)...)
	if quote == 1 {
		b = append(b, '\'')
	}
	return b, nil
}

// Define a variable of the structure type so that the compiler warns about
// noncompliance with the *serializer* interface.
var _ types.ValueScanner = (*ColonSepHexStr)(nil)

// Add colons to a hex string between every pair of digits.
func addColons(input string) string {
	builder := strings.Builder{}
	for idx, rune := range input {
		if idx >= 2 && idx%2 == 0 {
			builder.WriteRune(':')
		}
		builder.WriteRune(rune)
	}
	return builder.String()
}

// ScanValue reads the value out of the [types.Reader], converts it to the correct
// format, and stores it in the receiver.
func (s *ColonSepHexStr) ScanValue(rd types.Reader, n int) error {
	if n <= 0 {
		s.String = ""
		return nil
	}

	tmp, err := rd.ReadFullTemp()
	if err != nil {
		return err
	}

	noPrefix, _ := strings.CutPrefix(string(tmp), "\\x")
	s.String = addColons(noPrefix)
	return nil
}

// Define a variable of the structure type so that the compiler warns about
// noncompliance with [json.Unmarshaler].
var _ json.Unmarshaler = (*ColonSepHexStr)(nil)

// UnmarshalJSON adds a [ColonSepHexStr] wrapper around a plain string value.
func (s *ColonSepHexStr) UnmarshalJSON(b []byte) error {
	var deserialized string
	if err := json.Unmarshal(b, &deserialized); err != nil {
		return err
	}

	s.String = deserialized
	return nil
}

// Define a variable of the structure type so that the compiler warns about
// noncompliance with [json.Marshaler].
var _ json.Marshaler = (*ColonSepHexStr)(nil)

// MarshalJSON serializes a [ColonSepHexStr] as a plain string.
func (s *ColonSepHexStr) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String)
}
