package bind9config

var _ formattedElement = (*Directory)(nil)

// Directory is the statement used to specify the absolute path prepended to all
// relative paths in the configuration.
type Directory struct {
	// Absolute path prepended to all relative paths in the configuration.
	Path string `parser:"@String"`
}

// Returns the serialized BIND 9 configuration for the directory statement.
func (d *Directory) getFormattedOutput(filter *Filter) formatterOutput {
	return newFormatterClausef(`directory "%s"`, d.Path)
}
