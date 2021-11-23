package dumper

import (
	"io"
	"time"

	"github.com/pkg/errors"
	dumperdumps "isc.org/stork/server/dumper/dumps"
	storkutil "isc.org/stork/util"
)

// Function that produces the names for the artifacts.
// It is expected to return unique name for each dump-artifact combination.
// The result haven't must be deterministic (e.g. may contain a timestamp).
type namingConvention func(dump dumperdumps.Dump, artifact dumperdumps.Artifact) string

// Serialize the Go object to the binary content. It is expected to
// return human-readable output (e.g. JSON or YAML).
type structSerializer func(interface{}) ([]byte, error)

// Save the dumps to binary content in any format.
// It is responsible for serialize the dump artifacts
// and design of exported structure.
type saver interface {
	Save(target io.Writer, dumps []dumperdumps.Dump) error
}

// Structure that saves the dumps to the tarball archive.
// Each dump artifact is located in a separate file.
type tarbalSaver struct {
	serializer       structSerializer
	namingConvention namingConvention
}

// To create the tarball saver you need to provide a serializer that specify the output format
// for the struct artifacts and a naming convention used to name the artifact files.
func newTarbalSaver(serializer structSerializer, namingConvention namingConvention) *tarbalSaver {
	return &tarbalSaver{
		serializer:       serializer,
		namingConvention: namingConvention,
	}
}

// Save the dumps as a tarball archive.
// Remember that the "target" writter position is at the end after finishing this process.
func (t *tarbalSaver) Save(target io.Writer, dumps []dumperdumps.Dump) error {
	tarbal := storkutil.NewTarballWriter(target)
	defer tarbal.Close()

	for _, dump := range dumps {
		for i := 0; i < dump.GetArtifactsNumber(); i++ {
			artifact := dump.GetArtifact(i)
			path := t.namingConvention(dump, artifact)

			var rawContent []byte
			switch a := artifact.(type) {
			case dumperdumps.StructArtifact:
				var err error
				rawContent, err = t.serializer(a.GetStruct())
				if err != nil {
					return errors.Wrapf(err, "cannot serialize a dump artifact: %s - %s", dump.GetName(), artifact.GetName())
				}
			case dumperdumps.BinaryArtifact:
				rawContent = a.GetBinary()
			default:
				return errors.Errorf("unknown type of artifact: %s - %s", dump.GetName(), artifact.GetName())
			}

			err := tarbal.AddContent(path, rawContent, time.Now().UTC())
			if err != nil {
				return errors.Wrapf(err, "cannot append a dump artifact: %s - %s to tarbal", dump.GetName(), artifact.GetName())
			}
		}
	}

	return nil
}
