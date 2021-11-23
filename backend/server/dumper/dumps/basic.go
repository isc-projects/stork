package dumps

// Trivial implementation of the dump interfaces.
// It should be enough in the most use cases.
// The artifacts are collected in the expected form
// and stored as an array.
//
// The artifacts can be assigned in the constructor (for trivial cases)
// or by appending to the internal array in the Execute function.
//
// The Execute method should be overwritten to define the execution code.
type BasicDump struct {
	name      string
	artifacts []Artifact
}

func NewBasicDump(name string, artifacts ...Artifact) *BasicDump {
	return &BasicDump{
		name,
		artifacts,
	}
}

func (d *BasicDump) GetName() string {
	return d.name
}

func (d *BasicDump) GetArtifactsNumber() int {
	return len(d.artifacts)
}

func (d *BasicDump) GetArtifact(i int) Artifact {
	return d.artifacts[i]
}

func (d *BasicDump) AppendArtifact(artifact Artifact) {
	d.artifacts = append(d.artifacts, artifact)
}

func (d *BasicDump) Execute() error {
	return nil
}

// Base, abstract structure of the artifacts. It stores
// the name of the artifact.
type BasicArtifact struct {
	name string
}

func NewBasicArtifact(name string) *BasicArtifact {
	return &BasicArtifact{name}
}

func (a *BasicArtifact) GetName() string {
	return a.name
}

// Simple artifact-wrapper for any Go object. The content
// is intendent to serialize then assigned object must
// be serializable.
type BasicStructArtifact struct {
	BasicArtifact
	conent interface{}
}

func NewBasicStructArtifact(name string, content interface{}) *BasicStructArtifact {
	return &BasicStructArtifact{
		*NewBasicArtifact(name),
		content,
	}
}

func (a *BasicStructArtifact) GetStruct() interface{} {
	return a.conent
}

func (a *BasicStructArtifact) SetStruct(content interface{}) {
	a.conent = content
}

// Simple artifact-wrapper for the byte array.
type BasicBinaryArtifact struct {
	BasicArtifact
	conent []byte
}

func NewBasicBinaryArtifact(name string, content []byte) *BasicBinaryArtifact {
	return &BasicBinaryArtifact{
		*NewBasicArtifact(name),
		content,
	}
}

func (a *BasicBinaryArtifact) GetBinary() []byte {
	return a.conent
}
