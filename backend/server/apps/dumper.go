package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// The component dumps the machine
// configurations and related artifacts
// to single archive.

type fileCreator func(filename string) (*os.File, error)

type dumpResult struct {
	DumpName string
	Error    error
}

// Dump the machine and related artifacts to the single archive.
// Return the archive file (opened for read) if preceded successfully,
// otherwise nil file and not-nil error.
func DumpMachine(db *pg.DB, connectedAgents agentcomm.ConnectedAgents, machineID int64) (*os.File, error) {
	m, err := dbmodel.GetMachineByID(db, machineID)
	if err != nil {
		return nil, err
	}

	rootDir := os.TempDir()
	defer os.RemoveAll(rootDir)
	fileCreator := prepareFileCreator(rootDir)
	dumpResults := []*dumpResult{}

	err = dumpMachineEntry(fileCreator, m)
	dumpResults = append(dumpResults, &dumpResult{DumpName: "machine-entry", Error: err})

	err = dumpEvents(fileCreator, db, machineID)
	dumpResults = append(dumpResults, &dumpResult{DumpName: "last-events", Error: err})

	err = dumpLogs(fileCreator, connectedAgents, m)
	dumpResults = append(dumpResults, &dumpResult{DumpName: "log-tails", Error: err})

	err = dumpSettings(fileCreator, db)
	dumpResults = append(dumpResults, &dumpResult{DumpName: "settings", Error: err})

	err = saveDumpSummary(fileCreator, dumpResults)
	if err != nil {
		return nil, err
	}

	archiveFilename := fmt.Sprintf("dump_%d-%s-%d.tar-*.gz", m.ID, m.Address, m.AgentPort)
	archiveFile, err := ioutil.TempFile("", archiveFilename)
	if err != nil {
		return nil, errors.Wrap(err, "could not create the archive file")
	}

	err = storkutil.SaveDirectoryContentToTarbal(rootDir, archiveFile)
	if err != nil {
		archiveFile.Close()
		_ = os.Remove(archiveFilename)
		return nil, err
	}

	_, err = archiveFile.Seek(0, 0)
	err = errors.Wrap(err, "could not reset file position")

	return archiveFile, err
}

func dumpMachineEntry(targetProducer fileCreator, machine *dbmodel.Machine) error {
	return saveToFile(targetProducer, "machine", machine)
}

func dumpEvents(targetProducer fileCreator, db *pg.DB, machineID int64) error {
	events, _, err := dbmodel.GetEventsByPage(db, 0, 0, 1, nil, nil, &machineID, nil, "", dbmodel.SortDirAny)
	if err != nil {
		return err
	}

	return saveToFile(targetProducer, "events", events)
}

func dumpSettings(targetProducer fileCreator, db *pg.DB) error {
	settings, err := dbmodel.GetAllSettings(db)
	if err != nil {
		return nil
	}

	return saveToFile(targetProducer, "settings", settings)
}

func dumpLogs(targetProducer fileCreator, connectedAgents agentcomm.ConnectedAgents, machine *dbmodel.Machine) error {
	for _, app := range machine.Apps {
		for _, daemon := range app.Daemons {
			for logTargetID, logTarget := range daemon.LogTargets {
				if logTarget.Output == "stdout" || logTarget.Output == "stderr" ||
					strings.HasPrefix(logTarget.Output, "syslog") {
					continue
				}

				contents, err := connectedAgents.TailTextFile(
					context.Background(),
					machine.Address,
					machine.AgentPort,
					logTarget.Output,
					4000)

				tail := &models.LogTail{
					Machine: &models.AppMachine{
						ID:       machine.ID,
						Address:  machine.Address,
						Hostname: machine.State.Hostname,
					},
					AppID:           app.ID,
					AppName:         app.Name,
					AppType:         app.Type,
					LogTargetOutput: logTarget.Output,
					Contents:        contents,
					Error:           err.Error(),
				}

				filename := fmt.Sprintf("log_m-%d-%s_a-%d-%s_d-%d-%s_t-%d-%s.log",
					machine.ID, machine.Address,
					app.ID, app.Name,
					daemon.ID, daemon.Name,
					logTargetID, logTarget.Name)
				err = saveToFile(targetProducer, filename, tail)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func prepareFileCreator(root string) fileCreator {
	now := time.Now().UTC()
	timestampPrefix := now.Format("2006-01-02T15-04-05_")
	return func(filename string) (*os.File, error) {
		filePath := timestampPrefix + path.Join(root, filename)
		return os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0o755)
	}
}

func saveDumpSummary(targetProducer fileCreator, results []*dumpResult) error {
	type entry struct {
		Step   string
		Status string
		Error  error
	}

	type dumpSummary struct {
		Timestamp string
		Dumps     []entry
	}

	entries := make([]entry, len(results))
	for idx, res := range results {
		status := "Success"
		if res.Error != nil {
			status = "Fail"
		}

		entries[idx] = entry{
			Step:   res.DumpName,
			Status: status,
			Error:  res.Error,
		}
	}

	summary := dumpSummary{
		Timestamp: time.Now().UTC().Format("2016-01-02T15:04:05 UTC"),
		Dumps:     entries,
	}

	return saveToFile(targetProducer, "summary", summary)
}

func saveToFile(targetProducer fileCreator, name string, data interface{}) error {
	file, err := targetProducer(name + ".json")
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "could not serialize data to JSON")
	}
	_, err = file.Write(content)
	return errors.Wrap(err, "could not write the JSON content")
}
