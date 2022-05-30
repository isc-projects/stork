package dump

// Trivial implementation of the dump interfaces.
// It should be enough in the most use cases.
// The artifacts are collected in the expected form
// and stored as an array.
//
// The Execute method should be overwritten to define the execution code.
type BasicDump struct {
	name      string
	artifacts []Artifact
}

// Create instance of the basic dump.
// The artifacts can be assigned in this constructor (for trivial cases)
// or by appending to the internal array in the AppendArtifact function.
func NewBasicDump(name string, artifacts ...Artifact) *BasicDump {
	return &BasicDump{
		name,
		artifacts,
	}
}

// The name of the dump.
// Default implementation returns the fixed name
// from the constructor.
func (d *BasicDump) GetName() string {
	return d.name
}

// Returns the length of the internal artifact slice.
func (d *BasicDump) GetArtifactsNumber() int {
	return len(d.artifacts)
}

// Returns the artifact from the internal artifact slice.
func (d *BasicDump) GetArtifact(i int) Artifact {
	return d.artifacts[i]
}

// Append artifact to the internal artifact slice.
func (d *BasicDump) AppendArtifact(artifact Artifact) {
	d.artifacts = append(d.artifacts, artifact)
}

// Default implementation does nothing and returns no error.
// It is good for trivial cases when the artifacts are provided
// in the constructor, but is should be overridden for the advanced
// use cases.
func (d *BasicDump) Execute() error {
	return nil
}

// Base, abstract structure of the artifacts. It stores
// the name of the artifact.
type BasicArtifact struct {
	name      string
	extension string
}

// Constructs the basic artifact. It shouldn't be used directly,
// but in the child class constructors.
func NewBasicArtifact(name, extension string) *BasicArtifact {
	return &BasicArtifact{name, extension}
}

// Returns a name provided in the constructor.
func (a *BasicArtifact) GetName() string {
	return a.name
}

// Returns an extension provided in the constructor.
func (a *BasicArtifact) GetExtension() string {
	return a.extension
}

// Simple artifact-wrapper for a Go object. The content
// must be serializable.
type BasicStructArtifact struct {
	BasicArtifact
	content interface{}
}

// Constructs the artifact with the Go object as content.
// The content must be serializable.
func NewBasicStructArtifact(name string, content interface{}) *BasicStructArtifact {
	return &BasicStructArtifact{
		*NewBasicArtifact(name, ".json"),
		content,
	}
}

// The content getter. Part of the StructArtifact interface.
func (a *BasicStructArtifact) GetStruct() interface{} {
	return a.content
}

// The content setter. It is useful in case when the artifact
// object is created before the content is ready.
func (a *BasicStructArtifact) SetStruct(content interface{}) {
	a.content = content
}

// Simple artifact-wrapper for the byte array.
type BasicBinaryArtifact struct {
	BasicArtifact
	content []byte
}

// Constructs the artifact with binary data as content.
func NewBasicBinaryArtifact(name, extension string, content []byte) *BasicBinaryArtifact {
	return &BasicBinaryArtifact{
		*NewBasicArtifact(name, extension),
		content,
	}
}

// The content getter. Part of the BinaryArtifact interface.
func (a *BasicBinaryArtifact) GetBinary() []byte {
	return a.content
}
