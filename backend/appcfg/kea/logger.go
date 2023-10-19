package keaconfig

// A structure representing a single logger configuration.
// Kea 2.5 introduced an alias output-options. Stork must now
// support both output_options and output-options and return
// combined loggers configuration.
type Logger struct {
	Name               string                `json:"name"`
	OutputOptions      []LoggerOutputOptions `json:"output_options"`
	OutputOptionsAlias []LoggerOutputOptions `json:"output-options"`
	Severity           string                `json:"severity"`
	DebugLevel         int                   `json:"debuglevel"`
}

// A structure representing output_options for a logger.
type LoggerOutputOptions struct {
	Output string `json:"output"`
}

// Returns combined OutputOptions and OutputOptionsAlias.
func (logger Logger) GetAllOutputOptions() []LoggerOutputOptions {
	return append(logger.OutputOptions, logger.OutputOptionsAlias...)
}
