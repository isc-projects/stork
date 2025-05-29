package kea

import (
	"context"
	"math/big"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Statistics puller is responsible for fetching the data using the Kea
// statistic hook.
type StatsPuller struct {
	*agentcomm.PeriodicPuller
	*RpsWorker
}

// Create a StatsPuller object that in background pulls Kea stats about leases.
// Beneath it spawns a goroutine that pulls stats periodically from Kea apps (that are stored in database).
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents) (*StatsPuller, error) {
	statsPuller := &StatsPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Stats puller", "kea_stats_puller_interval",
		statsPuller.pullStats)
	if err != nil {
		return nil, err
	}
	statsPuller.PeriodicPuller = periodicPuller

	// Create RpsWorker instance
	rpsWorker, err := NewRpsWorker(db)
	if err != nil {
		return nil, err
	}
	statsPuller.RpsWorker = rpsWorker

	return statsPuller, nil
}

// Shutdown StatsPuller. It stops goroutine that pulls stats.
func (statsPuller *StatsPuller) Shutdown() {
	statsPuller.PeriodicPuller.Shutdown()
}

// Pull stats periodically for all Kea apps which Stork is monitoring. The function returns
// last encountered error.
func (statsPuller *StatsPuller) pullStats() error {
	// get list of all kea apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.DB, dbmodel.AppTypeKea)
	if err != nil {
		return err
	}

	// get lease stats from each kea app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("Error occurred while getting stats from app %d: %+v", dbApp.ID, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("Completed pulling lease stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))

	// estimate addresses utilization for subnets
	subnets, err := dbmodel.GetSubnetsWithLocalSubnets(statsPuller.DB)
	if err != nil {
		return err
	}

	if len(subnets) == 0 {
		return lastErr
	}

	counter := newStatisticsCounter()

	// The total IPv4 and IPv6 addresses statistics returned by Kea exclude
	// out-of-pool reservations, yielding possibly incorrect utilization.
	// The utilization can be corrected by including the out-of-pool
	// reservation counts from the Stork database.
	outOfPoolCounters, err := dbmodel.CountOutOfPoolAddressReservations(statsPuller.DB)
	if err != nil {
		return err
	}
	counter.setOutOfPoolAddresses(outOfPoolCounters)

	outOfPoolCounters, err = dbmodel.CountOutOfPoolPrefixReservations(statsPuller.DB)
	if err != nil {
		return err
	}
	counter.setOutOfPoolPrefixes(outOfPoolCounters)

	// Assume that all global reservations are out-of-pool for all subnets.
	outOfPoolGlobalIPv4Addresses, outOfPoolGlobalIPv6Addresses, outOfPoolGlobalDelegatedPrefixes, err := dbmodel.CountGlobalReservations(statsPuller.DB)
	if err != nil {
		return err
	}

	counter.global.totalIPv4Addresses.AddUint64(outOfPoolGlobalIPv4Addresses)
	counter.global.totalIPv6Addresses.AddUint64(outOfPoolGlobalIPv6Addresses)
	counter.global.totalDelegatedPrefixes.AddUint64(outOfPoolGlobalDelegatedPrefixes)

	// The HA servers share the same lease database and return the same
	// statistics. The statistics from the passive daemons are excluded from
	// calculations to avoid counting the same lease multiple times. The
	// calculator uses only the active daemon statistics because the active
	// daemon's database is overriding others.
	excludedDaemons, err := dbmodel.GetPassiveHADaemonIDs(statsPuller.DB)
	if err != nil {
		return err
	}
	counter.setExcludedDaemons(excludedDaemons)

	// go through all Subnets and:
	// 1) estimate utilization per Subnet and per SharedNetwork
	// 2) estimate global stats
	for _, sn := range subnets {
		su := counter.add(sn)
		err = sn.UpdateStatistics(
			statsPuller.DB,
			su,
		)
		if err != nil {
			lastErr = err
			log.Errorf("Cannot update utilization (%.3f, %.3f) in subnet %d: %s",
				su.GetAddressUtilization(), su.GetDelegatedPrefixUtilization(), sn.ID, err)
			continue
		}
	}

	// shared network utilization
	for sharedNetworkID, u := range counter.sharedNetworks {
		err = dbmodel.UpdateStatisticsInSharedNetwork(
			statsPuller.DB, sharedNetworkID, u,
		)
		if err != nil {
			lastErr = err
			log.Errorf("Cannot update utilization (%.3f, %.3f) in shared network %d: %s",
				u.GetAddressUtilization(), u.GetDelegatedPrefixUtilization(), sharedNetworkID, err)
			continue
		}
	}

	// global stats to collect
	statsMap := map[dbmodel.SubnetStatsName]*big.Int{
		dbmodel.SubnetStatsNameTotalAddresses:    counter.global.totalIPv4Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedAddresses: counter.global.totalAssignedIPv4Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedAddresses: counter.global.totalDeclinedIPv4Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameTotalNAs:          counter.global.totalIPv6Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedNAs:       counter.global.totalAssignedIPv6Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameDeclinedNAs:       counter.global.totalDeclinedIPv6Addresses.ToBigInt(),
		dbmodel.SubnetStatsNameAssignedPDs:       counter.global.totalAssignedDelegatedPrefixes.ToBigInt(),
		dbmodel.SubnetStatsNameTotalPDs:          counter.global.totalDelegatedPrefixes.ToBigInt(),
	}

	// update global statistics in db
	err = dbmodel.SetStats(statsPuller.DB, statsMap)
	if err != nil {
		lastErr = err
	}

	return lastErr
}

// A key that is used in map that is mapping from (local subnet id, inet family) to LocalSubnet struct.
type localSubnetKey struct {
	LocalSubnetID int64
	Family        int
}

// Processes statistics from the `statistic-get-all` response for the given daemon.
func (statsPuller *StatsPuller) storeDaemonStats(response keactrl.StatisticGetAllResponseItem, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	var lastErr error
	err := statsPuller.storeSubnetStats(response.Arguments, subnetsMap, dbApp, family)
	if err != nil {
		log.WithError(err).Error("Error handling subnet statistics")
		lastErr = err
	}

	err = statsPuller.storeAddressPoolStats(response.Arguments, subnetsMap, dbApp, family)
	if err != nil {
		log.WithError(err).Error("Error handling address pool statistics")
		lastErr = err
	}

	err = statsPuller.storePrefixPoolStats(response.Arguments, subnetsMap, dbApp, family)
	if err != nil {
		log.WithError(err).Error("Error handling prefix pool statistics")
		lastErr = err
	}

	return lastErr
}

// Processes statistics from the given command response for subnets belonging to the daemon.
func (statsPuller *StatsPuller) storeSubnetStats(response []keactrl.StatisticGetAllResponseSample, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	var lastErr error

	statisticsPerSubnet := make(map[int64][]keactrl.StatisticGetAllResponseSample)
	for _, statEntry := range response {
		subnetID := statEntry.SubnetID
		if statEntry.IsSubnetSample() {
			statisticsPerSubnet[subnetID] = append(statisticsPerSubnet[subnetID], statEntry)
		}
	}

	for subnetID, statEntries := range statisticsPerSubnet {
		stats := dbmodel.SubnetStats{}
		subnet := subnetsMap[localSubnetKey{subnetID, family}]
		if subnet == nil {
			lastErr = errors.Errorf(
				"cannot find LocalSubnet for app: %d, local subnet ID: %d, family: %d",
				dbApp.ID, subnetID, family,
			)
			log.Error(lastErr.Error())
			continue
		}

		for _, statEntry := range statEntries {
			// Store the value as a best fit type to preserve compatibility
			// with the existing code. Some features expect the IPv4
			// statistics to be always stored as uint64, while IPv6 can be
			// uint64 or big int.
			stats.SetBigCounter(
				statEntry.Name,
				storkutil.NewBigCounterFromBigInt(
					statEntry.Value,
				),
			)
		}

		err := subnet.UpdateStats(statsPuller.DB, stats)
		if err != nil {
			log.Errorf(
				"Problem updating Kea stats for local subnet ID %d, app ID %d: %s",
				subnet.LocalSubnetID, dbApp.ID, err.Error(),
			)
			lastErr = err
		}
	}
	return lastErr
}

// Defines a common interface for address and prefix pools to update their
// statistics.
type measurablePools interface {
	UpdateStats(dbi dbops.DBI, stats dbmodel.SubnetStats) error
	GetKeaParameters() *keaconfig.PoolParameters
}

// Process statistics from the given command response for address pools
// belonging to the daemon.
func (statsPuller *StatsPuller) storeAddressPoolStats(response []keactrl.StatisticGetAllResponseSample, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	return statsPuller.storePoolStats(
		response, subnetsMap, dbApp, family,
		func(ls *dbmodel.LocalSubnet) []measurablePools {
			pools := make([]measurablePools, len(ls.AddressPools))
			for i := 0; i < len(ls.AddressPools); i++ {
				pools[i] = &ls.AddressPools[i]
			}
			return pools
		},
		func(sample keactrl.StatisticGetAllResponseSample) bool {
			return sample.IsAddressPoolSample()
		},
	)
}

// Process statistics from the given command response for delegated prefix
// pools belonging to the daemon.
func (statsPuller *StatsPuller) storePrefixPoolStats(response []keactrl.StatisticGetAllResponseSample, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	return statsPuller.storePoolStats(
		response, subnetsMap, dbApp, family,
		func(ls *dbmodel.LocalSubnet) []measurablePools {
			pools := make([]measurablePools, len(ls.PrefixPools))
			for i := 0; i < len(ls.PrefixPools); i++ {
				pools[i] = &ls.PrefixPools[i]
			}
			return pools
		},
		func(sample keactrl.StatisticGetAllResponseSample) bool {
			return sample.IsPrefixPoolSample()
		},
	)
}

// Process statistics from the given command response for pools belonging to
// the daemon.
//
// It is a generic function that handles any pool. It accepts a pool
// accessor that defines how to extract the pools from the local subnet, and a
// predicate specifies how to select statistic samples corresponding to the
// pools.
func (statsPuller *StatsPuller) storePoolStats(
	response []keactrl.StatisticGetAllResponseSample,
	subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet,
	dbApp *dbmodel.App, family int,
	poolAccessor func(*dbmodel.LocalSubnet) []measurablePools,
	statPredicate func(keactrl.StatisticGetAllResponseSample) bool,
) error {
	var lastErr error

	statisticsPerSubnetAndPool := make(map[int64]map[int64][]keactrl.StatisticGetAllResponseSample)
	for _, statEntry := range response {
		if !statPredicate(statEntry) {
			continue
		}

		if _, ok := statisticsPerSubnetAndPool[statEntry.SubnetID]; !ok {
			statisticsPerSubnetAndPool[statEntry.SubnetID] = make(map[int64][]keactrl.StatisticGetAllResponseSample)
		}

		poolID := *statEntry.GetPoolID()
		statisticsPerSubnetAndPool[statEntry.SubnetID][poolID] = append(
			statisticsPerSubnetAndPool[statEntry.SubnetID][poolID],
			statEntry,
		)
	}

	for subnetID, statisticsPerPool := range statisticsPerSubnetAndPool {
		subnet := subnetsMap[localSubnetKey{subnetID, family}]
		if subnet == nil {
			lastErr = errors.Errorf(
				"cannot find LocalSubnet for app: %d, local subnet ID: %d, family: %d",
				dbApp.ID, subnetID, family,
			)
			log.Error(lastErr.Error())
			continue
		}

		pools := poolAccessor(subnet)

		for statPoolID, statEntries := range statisticsPerPool {
			for _, pool := range pools {
				dbPoolID := pool.GetKeaParameters().PoolID
				if dbPoolID != statPoolID {
					continue
				}

				stats := dbmodel.SubnetStats{}

				for _, statEntry := range statEntries {
					// Store the value as a best fit type to preserve compatibility
					// with the existing code. Some features expect the IPv4
					// statistics to be always stored as uint64, while IPv6 can be
					// uint64 or big int.
					stats.SetBigCounter(
						statEntry.Name,
						storkutil.NewBigCounterFromBigInt(
							statEntry.Value,
						),
					)
				}

				err := pool.UpdateStats(statsPuller.DB, stats)
				if err != nil {
					log.Errorf(
						"Problem updating Kea stats for address pool ID %d, app ID %d: %s",
						statPoolID, dbApp.ID, err.Error(),
					)
					lastErr = err
				}
			}
		}
	}
	return lastErr
}

func (statsPuller *StatsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// If no dhcp daemons found then exit.
	if len(dbApp.GetActiveDHCPDaemonNames()) == 0 {
		return nil
	}

	// If we're running RPS, age off obsolete RPS data.
	if statsPuller.RpsWorker != nil {
		_ = statsPuller.RpsWorker.AgeOffRpsIntervals()
	}

	// Slices for tracking commands, the daemons they're sent to, and the responses
	cmds := []*keactrl.Command{}
	cmdDaemons := []*dbmodel.Daemon{}
	var responsesAny []any

	// Iterate over active daemons, adding commands and response containers
	// for dhcp4 and dhcp6 daemons.
	for _, d := range dbApp.Daemons {
		if d.KeaDaemon == nil || !d.Active {
			continue
		}
		if d.Name != dbmodel.DaemonNameDHCPv4 && d.Name != dbmodel.DaemonNameDHCPv6 {
			continue
		}

		cmdDaemons = append(cmdDaemons, d)
		cmds = append(cmds, keactrl.NewCommandBase(keactrl.StatisticGetAll, d.Name))
		responsesAny = append(responsesAny, &keactrl.StatisticGetAllResponse{})
	}

	// If there are no commands, nothing to do.
	if len(cmds) == 0 {
		return nil
	}

	// Forward commands to Kea.
	ctx := context.Background()

	var serialCmds []keactrl.SerializableCommand
	for _, cmd := range cmds {
		serialCmds = append(serialCmds, cmd)
	}

	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp, serialCmds, responsesAny...)
	if err != nil {
		return err
	}

	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	responseItems := make([]keactrl.StatisticGetAllResponseItem, len(responsesAny))
	for i := 0; i < len(responsesAny); i++ {
		response, ok := responsesAny[i].(*keactrl.StatisticGetAllResponse)
		if !ok {
			// This should never happen.
			return errors.Errorf("response is not of type StatisticGetAllResponse: %T", responsesAny[i])
		}

		if len(*response) != 1 {
			// Each request is sent to a single daemon.
			return errors.Errorf("too many entries in the response")
		}

		responseItem := (*response)[0]

		err := responseItem.GetError()
		if err != nil {
			return errors.WithMessage(err, "the statistic-get-all command returned an error")
		}

		if responseItem.Arguments == nil {
			return errors.Errorf("arguments missing in the statistic-get-all response")
		}

		responseItems[i] = responseItem
	}

	// Process the response for each command for each daemon.
	return statsPuller.processAppResponses(dbApp, cmds, cmdDaemons, responseItems)
}

// Iterates through the commands for each daemon and processes the command responses
// Was part of getStatsFromApp() until lint:backend complained about cognitive complexity.
func (statsPuller *StatsPuller) processAppResponses(dbApp *dbmodel.App, cmds []*keactrl.Command, cmdDaemons []*dbmodel.Daemon, responses []keactrl.StatisticGetAllResponseItem) error {
	// Check if we have the same number of commands and responses.
	if len(cmds) != len(responses) {
		return errors.Errorf("number of commands (%d) does not match number of responses (%d)", len(cmds), len(responses))
	}

	// Lease statistic processing needs app's local subnets
	subnets, err := dbmodel.GetAppLocalSubnets(statsPuller.DB, dbApp.ID)
	if err != nil {
		return err
	}

	// prepare a map that will speed up looking for LocalSubnet
	// based on local subnet id and inet family
	subnetsMap := make(map[localSubnetKey]*dbmodel.LocalSubnet)
	for _, sn := range subnets {
		family := sn.Subnet.GetFamily()
		subnetsMap[localSubnetKey{sn.LocalSubnetID, family}] = sn
	}

	var lastErr error
	for idx := 0; idx < len(cmds); idx++ {
		family := 4
		if cmdDaemons[idx].Name == dhcp6 {
			family = 6
		}

		response := responses[idx]

		err = statsPuller.storeDaemonStats(response, subnetsMap, dbApp, family)
		if err != nil {
			log.WithError(err).Error("Error handling subnet statistics  in " +
				"the statistic-get-all response")
			lastErr = err
		}

		err = statsPuller.RpsWorker.Response4Handler(cmdDaemons[idx], response)
		if err != nil {
			log.WithError(err).Error("Error handling RPS DHCPv4 statistics " +
				" in the statistic-get-all response")
			lastErr = err
		}

		err = statsPuller.RpsWorker.Response6Handler(cmdDaemons[idx], response)
		if err != nil {
			log.WithError(err).Error("Error handling RPS DHCPv6 statistics " +
				" in the statistic-get-all response")
			lastErr = err
		}
	}

	return lastErr
}
