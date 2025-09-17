package bind9config

import "strings"

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
	values[len(values)-1] = strings.TrimSuffix(values[len(values)-1], "//@stork:no-parse:")
	*c = RawContents(strings.Join(values, " "))
	return nil
}

// Returns the string representation of the unparsed contents.
func (c *RawContents) GetString() string {
	if c != nil {
		return string(*c)
	}
	return ""
}
