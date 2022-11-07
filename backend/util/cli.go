package storkutil

// The CLI flag that accepts an optional string value. If not provided
// the default one will be used.
type OptionalStringFlag struct {
	value string
}

// Constructs the optional string flag.
func NewOptionalStringFlag(defaultValue string) *OptionalStringFlag {
	return &OptionalStringFlag{
		value: defaultValue,
	}
}

// Implements the cli.Generic interface. Sets a flag value if non-empty
// string provided. There is a workaround for the cli package internals that
// only boolean flag can accept no arguments.
func (f *OptionalStringFlag) Set(value string) error {
	if value != "true" && value != "" {
		f.value = value
	}
	return nil
}

// Implements the cli.Generic interface. Prints the value.
func (f *OptionalStringFlag) String() string {
	return f.value
}

// Implements the Boolean flag interface because only boolean flag can accept
// no arguments.
func (f *OptionalStringFlag) IsBoolFlag() bool {
	return true
}
