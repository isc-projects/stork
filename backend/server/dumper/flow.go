package dumper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
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
		dbmodel.MachineRelationDaemonHAServices,
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
	return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}

// Naming convention: [DUMP_NAME]_[ARTIFACT_NAME]_[TIMESTAMP].[EXT] .
func flatStructureWithTimestampNamingConvention(dumpObj dump.Dump, artifact dump.Artifact) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	filename := fmt.Sprintf("%s_%s_%s%s", dumpObj.GetName(), artifact.GetName(),
		timestamp, artifact.GetExtension())
	// Remove the insane characters
	filename = strings.ReplaceAll(filename, "/", "?")
	filename = strings.ReplaceAll(filename, "*", "?")
	return filename
}

// Serialize a Go struct to pretty indented JSON without escaping characters
// problematic for HTML.
func indentJSONSerializer(v interface{}) (output []byte, err error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "    ")
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(v)
	if err == nil {
		output = buffer.Bytes()
	}
	return
}
