package dump

// Dump - single unit of the dump process.
// It may contain multiple result artifacts
// collected.
type Dump interface {
	// The name of the dump. It must return valid name
	// before execution and after failed execution.
	GetName() string
	// This function executes the dump and
	// produces the artifacts. Returns error
	// when the dump execution failed.
	// Prefer to call this once per instance.
	Execute() error

	// Returns number of produced artifacts.
	GetArtifactsNumber() int
	// Returns the artifact instance at specific position.
	// The result can be casted to specific type.
	// It may panic if the argument is less then 0 or
	// greater or equals to the artifacts number.
	GetArtifact(int) Artifact
}

// The portion of data collected during the dump.
type Artifact interface {
	// Return the artifact number.
	GetName() string
	// Return an expected artifact extension.
	GetExtension() string
}

// The artifact that contains pure (serializable)
// GoLang structure.
type StructArtifact interface {
	Artifact
	// Returns plain GoLang object. It must be
	// serializable.
	GetStruct() interface{}
}

// The artifact that contains raw bytes
// or binary content (e.g. file).
type BinaryArtifact interface {
	Artifact
	// Returns binary representation of the artifact.
	GetBinary() []byte
}
