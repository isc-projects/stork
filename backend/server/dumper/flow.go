package dumper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/go-pg/pg/v9"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dump"
)

// The main function of this module. It dumps the specific machine (and related data) to the tarball archive.
// Returns closeable stream with the dump binary and error. If the machine doesn't exist it returns
// nil and no error.
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
		return nil, nil
	}

	// Factory will create the dump instances
	factory := newFactory(db, m, connectedAgents)
	// Saver will save the dumps to the tarball as JSON and raw binary files
	// It uses a flat structure - it means the output doesn't contain subfolders.
	saver := newTarballSaver(indentJSONSerializer, flatStructureWithTimestampNamingConvention)

	// Init dump objects
	dumps := factory.createAll()
	// Perform dump process
	summary := executeDumps(dumps)
	// Include only successful dumps
	// The dump summary is one of the dump artifacts too.
	// Exact summary isn't returned to UI in the current version.
	dumps = summary.getSuccessfulDumps()

	// Save the results to auto-release container.
	return saveDumpsToAutoReleaseContainer(saver, dumps)
}

// Save the dumps to self-cleaned container. After the call to the Close function
// on the returned reader all resources will be released.
// The returned reader is ready to read.
func saveDumpsToAutoReleaseContainer(saver saver, dumps []dump.Dump) (io.ReadCloser, error) {
	// Prepare the temporary buffer.
	var buffer bytes.Buffer
	err := saver.Save(&buffer, dumps)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}

// Naming convention rules:
// 1. Filename starts with a timestamp.
// 2. Struct artifact ends with the JSON extension.
//    The binary artifacts ends with the artifact name (it may contain extension).
// 3. Naming convention doesn't use subfolders.
// 4. Filename contains the dump name and artifact name.
func flatStructureWithTimestampNamingConvention(dumpObj dump.Dump, artifact dump.Artifact) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	extension := ".json"
	if _, ok := artifact.(dump.BinaryArtifact); ok {
		extension = ""
	}
	filename := fmt.Sprintf("%s_%s_%s%s", timestamp, dumpObj.GetName(), artifact.GetName(), extension)
	// Remove the insane characters
	filename = strings.ReplaceAll(filename, "/", "?")
	filename = strings.ReplaceAll(filename, "*", "?")
	return filename
}

// Serialize Go struct to pretty indent JSON.
func indentJSONSerializer(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "    ")
}
