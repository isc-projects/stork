package dumps

// Dump - single unit of the dump process.
// It may contain multiple result artifacts
// collected.
type Dump interface {
	GetName() string
	Execute() error

	GetArtifactsNumber() int
	GetArtifact(int) Artifact
}

// The portion of data collected during the dump.
type Artifact interface {
	GetName() string
}

// The artifact that contains pure (serializable)
// GoLang structure.
type StructArtifact interface {
	Artifact
	GetStruct() interface{}
}

// The artifact that contains raw binary. It is
// intendent to store the files.
type BinaryArtifact interface {
	Artifact
	GetBinary() []byte
}
