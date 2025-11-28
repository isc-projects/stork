package bind9config

import "strings"

var (
	_ formattedElement = (*NoParse)(nil)
	_ formattedElement = (*NoParseScope)(nil)
	_ formattedElement = (*NoParseGlobal)(nil)
)

// A Stork-specific annotation to skip parsing statements between the
// @stork:no-parse:scope and @stork:no-parse:end directives, or after
// the @stork:no-parse:global directive.
type NoParse struct {
	NoParseScope  *NoParseScope  `parser:"( @@"`
	NoParseGlobal *NoParseGlobal `parser:"| @@ )"`
}

// Checks if the @stork:no-parse:global directive was used.
func (n *NoParse) IsGlobal() bool {
	return n.NoParseGlobal != nil
}

// Returns the unparsed contents within the @stork:no-parse:scope
// and @stork:no-parse:end directives, or after the @stork:no-parse:global
// directive.
func (n *NoParse) GetContentsString() string {
	switch {
	case n.NoParseScope != nil:
		return n.NoParseScope.Contents.GetString()
	case n.NoParseGlobal != nil:
		return n.NoParseGlobal.Contents.GetString()
	default:
		return ""
	}
}

// Returns the serialized contents of the @stork:no-parse:scope/@stork:no-parse:end directives.
func (n *NoParse) getFormattedOutput(filter *Filter) formatterOutput {
	return getFormatterClauseFromStruct(n, filter)
}

// Represents the @stork:no-parse:scope/@stork:no-parse:end directives.
type NoParseScope struct {
	Preamble string      `parser:"@NoParseScope"`
	Contents RawContents `parser:"@NoParseContents"`
	End      string      `parser:"@NoParseEnd"`
}

// Returns the serialized BIND 9 configuration for the @stork:no-parse:scope/@stork:no-parse:end directives.
func (n *NoParseScope) getFormattedOutput(filter *Filter) formatterOutput {
	builder := strings.Builder{}
	builder.WriteString("//@stork:no-parse:scope\n")
	builder.WriteString(n.Contents.GetString())
	builder.WriteString("\n//@stork:no-parse:end")
	return newFormatterLines(builder.String())
}

// Represents the @stork:no-parse:global directive.
type NoParseGlobal struct {
	Preamble string      `parser:"@NoParseGlobal"`
	Contents RawContents `parser:"@NoParseGlobalContents"`
}

// Returns the serialized contents of the @stork:no-parse:global directive.
func (n *NoParseGlobal) getFormattedOutput(filter *Filter) formatterOutput {
	builder := strings.Builder{}
	builder.WriteString("//@stork:no-parse:global\n")
	builder.WriteString(n.Contents.GetString())
	return newFormatterLines(builder.String())
}
