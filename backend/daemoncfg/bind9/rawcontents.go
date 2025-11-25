package bind9config

import (
	"strings"
	"unicode"
)

// Unparsed contents between the @stork:no-parse:scope and @stork:no-parse:end
// directives, or after the @stork:no-parse:global directive.
type RawContents string

// Captures the unparsed contents between the @stork:no-parse:scope
// and @stork:no-parse:end directives and removes the trailing
// @stork:no-parse: suffix which is appended by the lexer.
func (c *RawContents) Capture(values []string) error {
	if len(values) == 0 {
		return nil
	}
	joinedValues := strings.Join(values, " ")
	trimmedValues := strings.TrimRightFunc(
		strings.TrimSuffix(joinedValues, "//@stork:no-parse:"),
		unicode.IsSpace,
	)
	*c = RawContents(trimmedValues)
	return nil
}

// Returns the string representation of the unparsed contents.
func (c *RawContents) GetString() string {
	if c != nil {
		return string(*c)
	}
	return ""
}
