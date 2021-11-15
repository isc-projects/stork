package dumper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dumps"
	storkutil "isc.org/stork/util"
)

var ErrNotFoundMachine error = errors.New("machine not found")

// The main function of this module. It dumps the specific machine (and related data) to the tarball archive.
func DumpMachine(db *pg.DB, connectedAgents agentcomm.ConnectedAgents, machineID int64) (io.ReadCloser, error) {
	m, err := dbmodel.GetMachineByIDWithRelations(db, machineID,
		dbmodel.MachineRelationApps,
		dbmodel.MachineRelationDaemons,
		dbmodel.MachineRelationKeaDaemons,
		dbmodel.MachineRelationBind9Daemons,
		dbmodel.MachineRelationDaemonLogTargets,
		dbmodel.MachineRelationAppAccessPoints,
		dbmodel.MachineRelationKeaDHCPConfigs,
	)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFoundMachine
	}

	// Factory will create the dump instances
	factory := newFactory(db, m, connectedAgents)
	// Saver will save the dumps to the tarball as JSON and raw binary files
	// It uses a flat structure - it means the output doesn't contain subfolders.
	saver := newTarbalSaver(indentJSONSerializer, flatStructureWithTimestampNamingConvention)

	// Init dump objects
	dumps := factory.All()
	// Perform dump process
	summary := executeDumps(dumps)
	// Exclude the success dumps
	// The dump summary is one of the dump artifacts too.
	// Exact summary isn't returned to UI in the current version.
	dumps = summary.GetSuccessfulDumps()

	// Save the results to auto-release container.
	return saveDumpsToAutoReleaseContainer(saver, dumps)
}

// Save the dumps to self-cleaned container. After call the Close function
// on returned reader all resources will be released.
// The returned reader is ready to read.
func saveDumpsToAutoReleaseContainer(saver saver, dumps []dumps.Dump) (io.ReadCloser, error) {
	// Prepare the temporary file for the dump.
	target, err := ioutil.TempFile("", "stork-dump-*")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an archive file")
	}
	// The dump file will closed by the HTTP middelware. Therefore, we
	// pack it into self-destroy wrapper to be sure that the file will
	// be deleted. It is the temporary file. It means that it will deleted even
	// if the app crashes.
	elfDestructWrapper := storkutil.NewSelfDestructFileWrapper(target)

	// Save the dumps to the tarball.
	bufferWriter := bufio.NewWriter(target)
	err = saver.Save(bufferWriter, dumps)

	if err != nil {
		elfDestructWrapper.Close()
		return nil, errors.Wrap(err, "cannot save the dumps")
	}

	err = bufferWriter.Flush()
	if err != nil {
		return nil, errors.Wrap(err, "cannot flush the dump content")
	}

	// Reset the file position to the beginning.
	_, err = target.Seek(0, io.SeekStart)
	if err != nil {
		elfDestructWrapper.Close()
		return nil, errors.Wrap(err, "cannot seek the file")
	}

	return elfDestructWrapper, nil
}

// Naming convention rules:
// 1. Filename starts with a timestamp.
// 2. Struct artifact ends with the JSON extension.
//    The binary artifacts ends with the artifact name (it may contain extension).
// 3. Naming convention doesn't use subfolders.
// 4. Filename contains the dump name and artifact name.
func flatStructureWithTimestampNamingConvention(dump dumps.Dump, artifact dumps.Artifact) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	extension := ".json"
	if _, ok := artifact.(dumps.BinaryArtifact); ok {
		extension = ""
	}
	filename := fmt.Sprintf("%s_%s_%s%s", timestamp, dump.Name(), artifact.Name(), extension)
	// Remove the insane characters
	filename = strings.ReplaceAll(filename, "/", "?")
	filename = strings.ReplaceAll(filename, "*", "?")
	return filename
}

// Serialize Go struct to pretty indent JSON
func indentJSONSerializer(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "    ")
}
