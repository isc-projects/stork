package bind9config

import "fmt"

// String holds a quoted or unquoted string. It is used to parse tokens that
// can be either be quoted strings or identifiers.
type String struct {
	Quoted   *string `parser:"@String"`
	Unquoted *string `parser:"| @Ident"`
}

// Returns the value without quotes. It returns an empty string if it is not set.
func (s *String) GetValue() string {
	switch {
	case s == nil:
		return ""
	case s.Quoted != nil:
		return *s.Quoted
	case s.Unquoted != nil:
		return *s.Unquoted
	default:
		return ""
	}
}

// Returns the value with quotes if original value was quoted.
// Returns unquoted value if original value was unquoted.
func (s *String) GetOriginalValue() string {
	switch {
	case s == nil:
		return ""
	case s.Quoted != nil:
		return fmt.Sprintf(`"%s"`, *s.Quoted)
	case s.Unquoted != nil:
		return *s.Unquoted
	default:
		return ""
	}
}
