package kea

import (
	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Uniquely identifies a cb_cmds config-backend target. Two daemons that share
// the same server tag and config database will have the same configBackendID
// and therefore receive only one remote-subnet*-set command with multiple
// server tags, instead of one command per daemon.
type configBackendID struct {
	DBName string
	DBHost string
	DBPort int
}

// Constructs a configBackendID for a local subnet's cb_cmds daemon.
// The daemon's config databases list must not be empty.
func buildConfigBackendID(daemon *dbmodel.Daemon) (configBackendID, error) {
	config := daemon.KeaDaemon.Config
	dbs := config.GetAllDatabases().Config
	if len(dbs) == 0 {
		return configBackendID{}, errors.Errorf(
			"daemon [%d] has libdhcp_cb_cmds loaded but no config databases configured",
			daemon.ID,
		)
	}
	db := dbs[0]
	return configBackendID{
		DBName: db.Name,
		DBHost: db.Host,
		DBPort: db.Port,
	}, nil
}

// Calls fn once per unique add-subnet target among the given local subnets:
//   - subnet_cmds daemons: fn is called for each daemon.
//   - cb_cmds daemons: fn is called only for the first daemon per unique
//     config-backend target (DBName + DBHost + DBPort), so that
//     daemons sharing the same config backend receive only one
//     remote-subnet*-set command.
//
// Local subnets whose daemon or Kea configuration is nil, or whose hook type
// is unrecognized, are silently skipped.
func forEachUniqueTarget(
	localSubnets []*dbmodel.LocalSubnet,
	fn func(ls *dbmodel.LocalSubnet, serverTags []string) error,
) error {
	configBackendGroups := map[configBackendID][]*dbmodel.Daemon{}
	seen := map[configBackendID]struct{}{}

	// Group daemons by config backend for cb_cmds.
	for _, ls := range localSubnets {
		hook, err := getHookForAlteringSubnets(ls.Daemon)
		if err != nil {
			continue
		}
		if hook != hookCbCmds {
			continue
		}
		key, err := buildConfigBackendID(ls.Daemon)
		if err != nil {
			continue
		}
		configBackendGroups[key] = append(configBackendGroups[key], ls.Daemon)
	}

	// Call fn for each unique target. For cb_cmds, only the first daemon per
	// config backend is considered a target.
	for _, ls := range localSubnets {
		hook, err := getHookForAlteringSubnets(ls.Daemon)
		if err != nil {
			continue
		}
		var serverTags []string
		if hook == hookCbCmds {
			// Skip if the config backend for this daemon has already been
			// processed.
			// Otherwise, collect server tags for all daemons sharing the same
			// config backend and mark the backend as seen.
			key, err := buildConfigBackendID(ls.Daemon)
			if err != nil {
				return err
			}
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			daemons := configBackendGroups[key]
			serverTagSet := make(map[string]struct{})
			for _, d := range daemons {
				serverTag := d.KeaDaemon.ServerTag
				if serverTag == "" {
					serverTag = "all"
				}
				serverTagSet[serverTag] = struct{}{}
			}
			for tag := range serverTagSet {
				serverTags = append(serverTags, tag)
			}
		}
		if err := fn(ls, serverTags); err != nil {
			return err
		}
	}
	return nil
}
