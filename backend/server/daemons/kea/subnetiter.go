package kea

import (
	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Uniquely identifies a cb_cmds config-backend target. Two HA-paired
// daemons that share the same server tag and config database will have
// the same configBackendID and therefore receive only one
// remote-subnet*-set command.
type configBackendID struct {
	ServerTag string
	DBName    string
	DBHost    string
	DBPort    int
}

// Constructs a configBackendID for a local subnet's cb_cmds daemon.
// The daemon's config databases list must not be empty.
func buildConfigBackendID(daemon *dbmodel.Daemon) (configBackendID, error) {
	cfg := daemon.KeaDaemon.Config
	serverTag := "all"
	if tag := cfg.GetServerTag(); tag != nil {
		serverTag = *tag
	}
	dbs := cfg.GetAllDatabases().Config
	if len(dbs) == 0 {
		return configBackendID{}, errors.Errorf(
			"daemon %s has libdhcp_cb_cmds loaded but no config databases configured",
			daemon.Name,
		)
	}
	db := dbs[0]
	return configBackendID{
		ServerTag: serverTag,
		DBName:    db.Name,
		DBHost:    db.Host,
		DBPort:    db.Port,
	}, nil
}

// Calls fn once per unique add-subnet target among the given local subnets:
//   - subnet_cmds daemons: fn is called for each daemon.
//   - cb_cmds daemons: fn is called only for the first daemon per unique
//     config-backend target (ServerTag + DBName + DBHost + DBPort), so that
//     HA-paired daemons sharing the same config backend receive only one
//     remote-subnet*-set command.
//
// Local subnets whose daemon or Kea configuration is nil, or whose hook type
// is unrecognized, are silently skipped.
func forEachUniqueTarget(
	localSubnets []*dbmodel.LocalSubnet,
	fn func(ls *dbmodel.LocalSubnet) error,
) error {
	seen := map[configBackendID]struct{}{}
	for _, ls := range localSubnets {
		hook, err := getSubnetHook(ls.Daemon)
		if err != nil {
			continue
		}
		if hook == hookCbCmds {
			key, err := buildConfigBackendID(ls.Daemon)
			if err != nil {
				return err
			}
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
		}
		if err := fn(ls); err != nil {
			return err
		}
	}
	return nil
}
