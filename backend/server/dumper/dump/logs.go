package dump

import (
	"context"
	"fmt"
	"strings"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	storkutil "isc.org/stork/util"
)

// The dump of all fetchable logs of the monitored apps.
// It means that it dumps the log tails from each log target
// related to the machine except stdin/stdout/syslog targets.
type LogsDump struct {
	BasicDump
	machine    *dbmodel.Machine
	logSources LogTailSource
}

// Log tail source - it corresponds to agentcomm.ConnectedAgents interface.
// It is needed to avoid the dependency cycle.
type LogTailSource interface {
	TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error)
}

// Constructs the log dump instance. It needs access to the log tail source
// (prefer ConnectedAgents) that is used to fetch the log content.
func NewLogsDump(machine *dbmodel.Machine, logSources LogTailSource) *LogsDump {
	return &LogsDump{
		*NewBasicDump("logs"),
		machine, logSources,
	}
}

// It iterates over all log targets for a specific machine
// (the log targets of each daemon for each app in the machine)
// and fetches the tail of each log.
// Each log target dump has attached the metadata and is dumped to a separate artifact.
//
// Current implementation excludes the non-file logs (stdout, stderr, syslog).
func (d *LogsDump) Execute() error {
	for _, app := range d.machine.Apps {
		for _, daemon := range app.Daemons {
			for logTargetID, logTarget := range daemon.LogTargets {
				if logTarget.Output == "stdout" || logTarget.Output == "stderr" ||
					strings.HasPrefix(logTarget.Output, "syslog") {
					continue
				}

				contents, err := d.logSources.TailTextFile(
					context.Background(),
					d.machine,
					logTarget.Output,
					40000)

				var errStr string
				if err != nil {
					errStr = err.Error()
				}

				tail := &models.LogTail{
					Machine: &models.AppMachine{
						ID:       d.machine.ID,
						Address:  d.machine.Address,
						Hostname: d.machine.State.Hostname,
					},
					AppID:           storkutil.Ptr(app.ID),
					AppName:         storkutil.Ptr(app.Name),
					AppType:         storkutil.Ptr(app.Type.String()),
					LogTargetOutput: storkutil.Ptr(logTarget.Output),
					Contents:        contents,
					Error:           errStr,
				}

				name := fmt.Sprintf("a-%d-%s_d-%d-%s_t-%d-%s",
					app.ID, app.Name,
					daemon.ID, daemon.Name,
					logTargetID, logTarget.Name)

				d.AppendArtifact(NewBasicStructArtifact(
					name, tail,
				))
			}
		}
	}

	return nil
}
