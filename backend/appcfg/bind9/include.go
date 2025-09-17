package bind9config

var _ formattedElement = (*Include)(nil)

// Include is the statement used to include another configuration file.
// The included file can be parsed and its configuration statements expand
// the parent configuration. The "include" statement has the following format:
//
// include <filename>;
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#include-directive.
type Include struct {
	// Included file path.
	Path string `parser:"@String"`
}

// Returns the serialized BIND 9 configuration for the include statement.
func (i *Include) getFormattedOutput(filter *Filter) formatterOutput {
	return newFormatterClausef(`include "%s"`, i.Path)
}
