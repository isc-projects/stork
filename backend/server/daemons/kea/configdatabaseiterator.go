package kea

import (
	"maps"
	"reflect"
	"slices"

	"github.com/pkg/errors"
	keaconfig "isc.org/stork/daemoncfg/kea"
	dbmodel "isc.org/stork/server/database/model"
)

// Uniquely identifies a cb_cmds config-backend target. Two daemons that share
// the same config database will have the same configBackendKey and therefore
// receive only one remote-subnet*-set command with multiple
// server tags, instead of one command per daemon.
type configBackendKey struct {
	DBName string
	DBHost string
	DBPort int
}

// Constructs a configBackendKey for a local subnet's cb_cmds daemon.
// The daemon's config databases list must not be empty.
func buildConfigBackendKey(daemon *dbmodel.Daemon) (configBackendKey, error) {
	config := daemon.KeaDaemon.Config
	dbs := config.GetAllDatabases().Config
	if len(dbs) == 0 {
		return configBackendKey{}, errors.Errorf(
			"daemon [%d] has libdhcp_cb_cmds loaded but no config databases configured",
			daemon.ID,
		)
	}
	db := dbs[0]
	return configBackendKey{
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
func forEachUniqueConfigSource(
	localSubnets []*dbmodel.LocalSubnet,
	fn func(ls *dbmodel.LocalSubnet, serverTags []string) error,
) error {
	configBackendGroups := map[configBackendKey][]*dbmodel.Daemon{}
	localSubnetReferenceIdxByConfigBackend := map[configBackendKey]int{}
	seen := map[configBackendKey]struct{}{}

	// Group daemons by config backend for cb_cmds.
	for i, ls := range localSubnets {
		hook := ls.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAlteringHookLibrary()
		if hook != keaconfig.SubnetAlteringHookLibraryCBCmds {
			continue
		}
		key, err := buildConfigBackendKey(ls.Daemon)
		if err == nil {
			referenceIdx, found := localSubnetReferenceIdxByConfigBackend[key]
			if !found {
				localSubnetReferenceIdxByConfigBackend[key] = i
			} else {
				// If another local subnet has already been associated with
				// this config backend, check that they are consistent.
				reference := localSubnets[referenceIdx]

				var inconsistentField string
				switch {
				case reference.LocalSubnetID != ls.LocalSubnetID:
					inconsistentField = "Local subnet ID"
				case !reflect.DeepEqual(reference.KeaParameters, ls.KeaParameters):
					inconsistentField = "Kea parameters"
				case reference.Hash != ls.Hash:
					inconsistentField = "DHCP options"
				case !reflect.DeepEqual(reference.PrefixPools, ls.PrefixPools):
					inconsistentField = "Prefix pools"
				case !reflect.DeepEqual(reference.AddressPools, ls.AddressPools):
					inconsistentField = "Address pools"
				case !reflect.DeepEqual(reference.UserContext, ls.UserContext):
					inconsistentField = "User context"
				}

				if inconsistentField != "" {
					return errors.Errorf(
						"daemons sharing config backend %s@%s:%d have inconsistent %s",
						key.DBName,
						key.DBHost,
						key.DBPort,
						inconsistentField,
					)
				}
			}

			configBackendGroups[key] = append(configBackendGroups[key], ls.Daemon)
		}
	}

	// Call fn for each unique target. For cb_cmds, only the first daemon per
	// config backend is considered a target.
	for _, ls := range localSubnets {
		hook := ls.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAlteringHookLibrary()
		var serverTags []string
		if hook == keaconfig.SubnetAlteringHookLibraryNone || hook == keaconfig.SubnetAlteringHookLibraryAmbiguous {
			continue
		}
		if hook == keaconfig.SubnetAlteringHookLibraryCBCmds {
			// Skip if the config backend for this daemon has already been
			// processed.
			// Otherwise, collect server tags for all daemons sharing the same
			// config backend and mark the backend as seen.
			key, err := buildConfigBackendKey(ls.Daemon)
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
				serverTag := "all"
				if d.KeaDaemon.ServerTag != nil {
					serverTag = *d.KeaDaemon.ServerTag
				}
				serverTagSet[serverTag] = struct{}{}
			}
			serverTags = slices.Collect(maps.Keys(serverTagSet))
		}
		if err := fn(ls, serverTags); err != nil {
			return err
		}
	}
	return nil
}
