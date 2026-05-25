package kea

import (
	"maps"
	"reflect"
	"slices"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	DBPort int64
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
		DBPort: int64(db.Port),
	}, nil
}

// Calls fn once per unique add-subnet target among the given local subnets:
//   - subnet_cmds daemons: fn is called for each daemon.
//   - cb_cmds daemons: fn is called only for the first daemon per unique
//     config-backend target (DBName + DBHost + DBPort), so that
//     daemons sharing the same config backend receive only one
//     remote-subnet*-set command.
//
// It calls the provided fn with all local subnets sharing the same config
// source (it always has a one item for subnet_cmds daemons, and may have
// multiple for cb_cmds daemons).
//
// It is guaranteed that the local subnets passed to fn has at least one
// item.
//
// Local subnets whose daemon or Kea configuration is nil, or whose hook type
// is unrecognized, are silently skipped.
func forEachUniqueConfigSource(
	localSubnets []*dbmodel.LocalSubnet,
	fn func(localSubnets []*dbmodel.LocalSubnet) error,
) error {
	localSubnetsByBackend := map[configBackendKey][]*dbmodel.LocalSubnet{}

	// Group daemons by config backend for cb_cmds.
	for _, ls := range localSubnets {
		hook := ls.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAndSharedNetworkAlteringHookLibrary()
		switch hook {
		case keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds:
			// For non-cb_cmds daemons, the config source is the daemon config,
			// they are not grouped.
			key := configBackendKey{DBPort: ls.DaemonID}
			localSubnetsByBackend[key] = []*dbmodel.LocalSubnet{ls}
		case keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds:
			key, err := buildConfigBackendKey(ls.Daemon)
			if err != nil {
				log.WithError(err).Warnf(
					"skipping local subnet [%d] because its daemon [%d] has invalid config database configuration",
					ls.ID,
					ls.DaemonID,
				)
				continue
			}

			backendLocalSubnets, ok := localSubnetsByBackend[key]
			if !ok {
				backendLocalSubnets = []*dbmodel.LocalSubnet{}
			}
			localSubnetsByBackend[key] = append(backendLocalSubnets, ls)
		default:
			continue
		}
	}

	for _, localSubnets := range localSubnetsByBackend {
		if err := fn(localSubnets); err != nil {
			return err
		}
	}
	return nil
}

// Calls fn once per unique add-subnet target among the given local subnets:
//   - subnet_cmds daemons: fn is called for each daemon.
//   - cb_cmds daemons: fn is called only for the first daemon per unique
//     config-backend target (DBName + DBHost + DBPort), so that
//     daemons sharing the same config backend receive only one
//     remote-subnet*-set command.
//
// It validates that all daemons sharing the same config backend have
// consistent configurations (Kea parameters, DHCP options, pools, user
// context). In case of inconsistency, an error is returned describing the
// first detected inconsistency.
//
// It calls the provided fn with the first local subnet for each unique target
// and a list of all server tags sharing that target.
//
// Local subnets whose daemon or Kea configuration is nil, or whose hook type
// is unrecognized, are silently skipped.
func forEachUniqueConsistentConfigSource(
	localSubnets []*dbmodel.LocalSubnet,
	fn func(ls *dbmodel.LocalSubnet, serverTags []string) error,
) error {
	return forEachUniqueConfigSource(localSubnets, func(localSubnets []*dbmodel.LocalSubnet) error {
		// Check consistency. Fail if any two daemons have inconsistent data.
		if len(localSubnets) > 1 {
			reference := localSubnets[0]
			for _, ls := range localSubnets[1:] {
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
						"daemons with IDs [%d] and [%d] sharing config backend have inconsistent %s",
						reference.DaemonID,
						ls.DaemonID,
						inconsistentField,
					)
				}
			}
		}

		// Collect unique server tags.
		serverTagsSet := make(map[string]struct{})
		for _, ls := range localSubnets {
			serverTag := "all"
			if ls.Daemon.KeaDaemon.ServerTag != nil {
				serverTag = *ls.Daemon.KeaDaemon.ServerTag
			}
			serverTagsSet[serverTag] = struct{}{}
		}
		serverTags := slices.Collect(maps.Keys(serverTagsSet))

		// Call fn for each unique target. For cb_cmds, only the first daemon per
		// config backend is considered a target.
		if err := fn(localSubnets[0], serverTags); err != nil {
			return err
		}
		return nil
	})
}
