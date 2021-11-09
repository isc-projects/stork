package dumps

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v9"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
)

// The dump of the all fetchable logs.
// It means that it dumps the log tails from each log target
// related to the machine except stdin/stdout/syslog targets.
type LogsDump struct {
	BasicDump
	db      *pg.DB
	machine *dbmodel.Machine
	agents  agentcomm.ConnectedAgents
}

func NewLogsDump(db *pg.DB, machine *dbmodel.Machine, agents agentcomm.ConnectedAgents) *LogsDump {
	return &LogsDump{
		*NewBasicDump("logs"),
		db, machine, agents,
	}
}

func (d *LogsDump) Execute() error {
	for _, app := range d.machine.Apps {
		for _, daemon := range app.Daemons {
			for logTargetID, logTarget := range daemon.LogTargets {
				if logTarget.Output == "stdout" || logTarget.Output == "stderr" ||
					strings.HasPrefix(logTarget.Output, "syslog") {
					continue
				}

				contents, err := d.agents.TailTextFile(
					context.Background(),
					d.machine.Address,
					d.machine.AgentPort,
					logTarget.Output,
					4000)
				if err != nil {
					return err
				}

				tail := &models.LogTail{
					Machine: &models.AppMachine{
						ID:       d.machine.ID,
						Address:  d.machine.Address,
						Hostname: d.machine.State.Hostname,
					},
					AppID:           app.ID,
					AppName:         app.Name,
					AppType:         app.Type,
					LogTargetOutput: logTarget.Output,
					Contents:        contents,
					Error:           err.Error(),
				}

				name := fmt.Sprintf("a-%d-%s_d-%d-%s_t-%d-%s",
					app.ID, app.Name,
					daemon.ID, daemon.Name,
					logTargetID, logTarget.Name)

				d.artifacts = append(d.artifacts, NewBasicStructArtifact(
					name, tail,
				))
			}
		}
	}

	return nil
}
