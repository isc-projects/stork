package keaconfig

// A structure representing a single logger configuration.
type Logger struct {
	Name          string                `json:"name"`
	OutputOptions []LoggerOutputOptions `json:"output_options"`
	Severity      string                `json:"severity"`
	DebugLevel    int                   `json:"debuglevel"`
}

// A structure representing output_options for a logger.
type LoggerOutputOptions struct {
	Output string
}
