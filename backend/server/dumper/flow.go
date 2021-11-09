package dumper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dumps"
	storkutil "isc.org/stork/util"
)

// The main function of this module. It dumps the specific machine (and related data) to the tarball archive.
func DumpMachine(db *pg.DB, connectedAgents agentcomm.ConnectedAgents, machineID int64) (io.ReadCloser, error) {
	m, err := dbmodel.GetMachineByID(db, machineID)
	if err != nil {
		return nil, err
	}

	// Factory will create the dump instances
	factory := newFactory(db, m, connectedAgents)
	// Saver will save the dumps to the tarball as JSON and raw binary files
	// It uses a flat structure - it means the output doesn't contain subfolders.
	saver := newTarbalSaver(json.Marshal, flatStructureWithTimestampNamingConvention)

	// Init dump objects
	dumps := factory.All()
	// Perform dump process
	summary := execute(dumps)
	// Exclude the success dumps
	// The dump summary is one of the dump artifacts too.
	// Exact summary isn't returned to UI in the current version.
	dumps = summary.GetSuccessDumps()

	// Prepare the temporary file for the dump.
	target, err := ioutil.TempFile("", archiveName(m))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create an archive filename for machine: %d", m.ID)
	}
	// The dump file will closed by the HTTP middelware. Therefore, we
	// pack it into self-destroy wrapper to be sure that the file will
	// be deleted. It is the temporary file. It means that it will deleted even
	// if the app crashes.
	selfDestructWrapper := storkutil.NewSelfDestoyFileWrapper(target)

	// Save the dumps to the tarball.
	err = saver.Save(bufio.NewWriter(target), dumps)

	if err != nil {
		selfDestructWrapper.Close()
		return nil, errors.Wrap(err, "cannot save the dumps")
	}

	// Reset the file position to the beginning.
	_, err = target.Seek(0, 0)
	if err != nil {
		selfDestructWrapper.Close()
		return nil, errors.Wrap(err, "cannot seek the file")
	}

	return selfDestructWrapper, nil
}

// Naming convention rules:
// 1. Filename starts with a timestamp.
// 2. Struct artifact ends with the JSON extension.
//    The binary artifacts ends with the artifact name (it may contain extension).
// 3. Naming convention doesn't use subfolders.
// 4. Filename contains the dump name and artifact name.
func flatStructureWithTimestampNamingConvention(dump dumps.Dump, artifact dumps.Artifact) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	extension := ".json"
	if _, ok := artifact.(dumps.BinaryArtifact); ok {
		extension = ""
	}
	return fmt.Sprintf("%s_%s_%s%s", timestamp, dump.Name(), artifact.Name(), extension)
}

// Tarball name convention. It starts with a timestamp, contains machine ID and address,
// includes the random characters and ends with proper extension.
func archiveName(machine *dbmodel.Machine) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	return fmt.Sprintf("%s_%d-%s-*.tar.gz", timestamp, machine.ID, machine.Address)
}
