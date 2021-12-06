package dumper

import (
	"io"
	"time"

	"github.com/pkg/errors"
	"isc.org/stork/server/dumper/dump"
	storkutil "isc.org/stork/util"
)

// Function that produces the names for the artifacts.
// It is expected to return unique name for each dump-artifact combination.
// The result haven't must be deterministic (e.g. may contain a timestamp).
type namingConvention func(dump dump.Dump, artifact dump.Artifact) string

// Serialize the Go object to the binary content. It is expected to
// return human-readable output (e.g. JSON or YAML).
type structSerializer func(interface{}) ([]byte, error)

// Save the dumps to binary content in any format.
// It is responsible for serialize the dump artifacts
// and design of exported structure.
type saver interface {
	Save(target io.Writer, dumps []dump.Dump) error
}

// Structure that saves the dumps to the tarball archive.
// Each dump artifact is located in a separate file.
type tarballSaver struct {
	serializer       structSerializer
	namingConvention namingConvention
}

// To create the tarball saver you need to provide a serializer that specify the output format
// for the struct artifacts and a naming convention used to name the artifact files.
func newTarballSaver(serializer structSerializer, namingConvention namingConvention) *tarballSaver {
	return &tarballSaver{
		serializer:       serializer,
		namingConvention: namingConvention,
	}
}

// Save the dumps as a tarball archive.
// Remember that the "target" writer position is at the end after finishing this process.
func (t *tarballSaver) Save(target io.Writer, dumps []dump.Dump) error {
	tarball := storkutil.NewTarballWriter(target)
	defer tarball.Close()

	for _, dumpObj := range dumps {
		for i := 0; i < dumpObj.GetArtifactsNumber(); i++ {
			artifact := dumpObj.GetArtifact(i)
			path := t.namingConvention(dumpObj, artifact)

			var rawContent []byte
			switch a := artifact.(type) {
			case dump.StructArtifact:
				var err error
				rawContent, err = t.serializer(a.GetStruct())
				if err != nil {
					return errors.Wrapf(err, "cannot serialize a dump artifact: %s - %s", dumpObj.GetName(), artifact.GetName())
				}
			case dump.BinaryArtifact:
				rawContent = a.GetBinary()
			default:
				return errors.Errorf("unknown type of artifact: %s - %s", dumpObj.GetName(), artifact.GetName())
			}

			err := tarball.AddContent(path, rawContent, time.Now().UTC())
			if err != nil {
				return errors.Wrapf(err, "cannot append a dump artifact: %s - %s to tarball", dumpObj.GetName(), artifact.GetName())
			}
		}
	}

	return nil
}
