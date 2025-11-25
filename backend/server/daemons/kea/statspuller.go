package kea

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
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
// Beneath it spawns a goroutine that pulls stats periodically from Kea daemons
// (that are stored in database).
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

// Pull stats periodically for all Kea daemons which Stork is monitoring. The function returns
// last encountered error.
func (statsPuller *StatsPuller) pullStats() error {
	// get list of all kea daemons from database
	daemons, err := dbmodel.GetDHCPDaemons(statsPuller.DB)
	if err != nil {
		return err
	}

	// get lease stats from each kea daemon
	var lastErr error
	daemonsOkCnt := 0
	for _, daemon := range daemons {
		err := statsPuller.getStatsFromDaemon(&daemon)
		if err != nil {
			lastErr = err
			log.WithError(err).Errorf("Error occurred while getting stats from daemon %d", daemon.ID)
		} else {
			daemonsOkCnt++
		}
	}
	log.Infof("Completed pulling lease stats from Kea daemons: %d/%d succeeded", daemonsOkCnt, len(daemons))

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
	outOfPoolAddressCounters, err := dbmodel.CountOutOfPoolAddressReservations(statsPuller.DB)
	if err != nil {
		return err
	}

	outOfPoolPrefixCounters, err := dbmodel.CountOutOfPoolPrefixReservations(statsPuller.DB)
	if err != nil {
		return err
	}

	// Assume that all global reservations are out-of-pool for all subnets.
	outOfPoolGlobalIPv4Addresses, outOfPoolGlobalIPv6Addresses, outOfPoolGlobalDelegatedPrefixes, err := dbmodel.CountGlobalReservations(statsPuller.DB)
	if err != nil {
		return err
	}

	counter.setOutOfPoolShifts(outOfPoolShifts{
		outOfPoolAddresses:       outOfPoolAddressCounters,
		outOfPoolPrefixes:        outOfPoolPrefixCounters,
		outOfPoolGlobalAddresses: outOfPoolGlobalIPv4Addresses,
		outOfPoolGlobalNAs:       outOfPoolGlobalIPv6Addresses,
		outOfPoolGlobalPrefixes:  outOfPoolGlobalDelegatedPrefixes,
	})

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
			log.WithError(err).Errorf("Cannot update utilization (%.3f, %.3f) in subnet %d",
				su.GetAddressUtilization(), su.GetDelegatedPrefixUtilization(), sn.ID)
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
			log.WithError(err).Errorf("Cannot update utilization (%.3f, %.3f) in shared network %d",
				u.GetAddressUtilization(), u.GetDelegatedPrefixUtilization(), sharedNetworkID)
			continue
		}
	}

	// global stats to collect
	statsMap := counter.GetStatistics()

	// update global statistics in db
	err = dbmodel.SetStats(statsPuller.DB, statsMap)
	if err != nil {
		lastErr = err
	}

	return lastErr
}

// Processes statistics from the `statistic-get-all` response for the given daemon.
func (statsPuller *StatsPuller) storeDaemonStats(response *keactrl.StatisticGetAllResponse, subnetsMap map[int64]*dbmodel.LocalSubnet, daemon *dbmodel.Daemon) error {
	var lastErr error
	err := statsPuller.storeStats(response.Arguments, subnetsMap, daemon)
	if err != nil {
		log.WithError(err).Error("Error handling subnet statistics")
		lastErr = err
	}

	err = statsPuller.storeAddressPoolStats(response.Arguments, subnetsMap, daemon)
	if err != nil {
		log.WithError(err).Error("Error handling address pool statistics")
		lastErr = err
	}

	err = statsPuller.storePrefixPoolStats(response.Arguments, subnetsMap, daemon)
	if err != nil {
		log.WithError(err).Error("Error handling prefix pool statistics")
		lastErr = err
	}

	return lastErr
}

// Processes statistics from the given command response for subnets belonging to the daemon.
func (statsPuller *StatsPuller) storeStats(response []*keactrl.StatisticGetAllResponseSample, subnetsMap map[int64]*dbmodel.LocalSubnet, daemon *dbmodel.Daemon) error {
	var lastErr error

	statisticsPerSubnet := make(map[int64][]*keactrl.StatisticGetAllResponseSample)
	for _, statEntry := range response {
		subnetID := statEntry.SubnetID
		if statEntry.IsSubnetSample() {
			statisticsPerSubnet[subnetID] = append(statisticsPerSubnet[subnetID], statEntry)
		}
	}

	for subnetID, statEntries := range statisticsPerSubnet {
		stats := dbmodel.Stats{}
		subnet := subnetsMap[subnetID]
		if subnet == nil {
			lastErr = errors.Errorf(
				"cannot find LocalSubnet for daemon: %d, local subnet ID: %d",
				daemon.ID, subnetID,
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
				"Problem updating Kea stats for local subnet ID %d, daemon ID %d: %s",
				subnet.LocalSubnetID, daemon.ID, err.Error(),
			)
			lastErr = err
		}
	}
	return lastErr
}

// Defines a common interface for address and prefix pools to update their
// statistics.
type measurablePools interface {
	UpdateStats(dbi dbops.DBI, stats dbmodel.Stats) error
	GetKeaParameters() *keaconfig.PoolParameters
}

// Process statistics from the given command response for address pools
// belonging to the daemon.
func (statsPuller *StatsPuller) storeAddressPoolStats(response []*keactrl.StatisticGetAllResponseSample, subnetsMap map[int64]*dbmodel.LocalSubnet, daemon *dbmodel.Daemon) error {
	return statsPuller.storePoolStats(
		response, subnetsMap, daemon,
		func(ls *dbmodel.LocalSubnet) []measurablePools {
			pools := make([]measurablePools, len(ls.AddressPools))
			for i := 0; i < len(ls.AddressPools); i++ {
				pools[i] = &ls.AddressPools[i]
			}
			return pools
		},
		func(sample *keactrl.StatisticGetAllResponseSample) bool {
			return sample.IsAddressPoolSample()
		},
	)
}

// Process statistics from the given command response for delegated prefix
// pools belonging to the daemon.
func (statsPuller *StatsPuller) storePrefixPoolStats(response []*keactrl.StatisticGetAllResponseSample, subnetsMap map[int64]*dbmodel.LocalSubnet, daemon *dbmodel.Daemon) error {
	return statsPuller.storePoolStats(
		response, subnetsMap, daemon,
		func(ls *dbmodel.LocalSubnet) []measurablePools {
			pools := make([]measurablePools, len(ls.PrefixPools))
			for i := 0; i < len(ls.PrefixPools); i++ {
				pools[i] = &ls.PrefixPools[i]
			}
			return pools
		},
		func(sample *keactrl.StatisticGetAllResponseSample) bool {
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
	response []*keactrl.StatisticGetAllResponseSample,
	subnetsMap map[int64]*dbmodel.LocalSubnet,
	daemon *dbmodel.Daemon,
	poolAccessor func(*dbmodel.LocalSubnet) []measurablePools,
	statPredicate func(*keactrl.StatisticGetAllResponseSample) bool,
) error {
	var lastErr error

	statisticsPerSubnetAndPool := make(map[int64]map[int64][]*keactrl.StatisticGetAllResponseSample)
	for _, statEntry := range response {
		if !statPredicate(statEntry) {
			continue
		}

		if _, ok := statisticsPerSubnetAndPool[statEntry.SubnetID]; !ok {
			statisticsPerSubnetAndPool[statEntry.SubnetID] = make(map[int64][]*keactrl.StatisticGetAllResponseSample)
		}

		poolID := *statEntry.GetPoolID()
		statisticsPerSubnetAndPool[statEntry.SubnetID][poolID] = append(
			statisticsPerSubnetAndPool[statEntry.SubnetID][poolID],
			statEntry,
		)
	}

	for subnetID, statisticsPerPool := range statisticsPerSubnetAndPool {
		subnet := subnetsMap[subnetID]
		if subnet == nil {
			lastErr = errors.Errorf(
				"cannot find LocalSubnet for daemon: %d, local subnet ID: %d",
				daemon.ID, subnetID,
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

				stats := dbmodel.Stats{}

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
					log.WithError(err).Errorf(
						"Problem updating Kea stats for address pool ID %d, daemon ID %d",
						statPoolID, daemon.ID,
					)
					lastErr = err
				}
			}
		}
	}
	return lastErr
}

func (statsPuller *StatsPuller) getStatsFromDaemon(daemon *dbmodel.Daemon) error {
	// If we're running RPS, age off obsolete RPS data.
	if statsPuller.RpsWorker != nil {
		_ = statsPuller.AgeOffRpsIntervals()
	}

	if daemon.KeaDaemon == nil || !daemon.Active {
		return nil
	}
	if !daemon.Name.IsDHCP() {
		return nil
	}

	cmd := keactrl.NewCommandBase(keactrl.StatisticGetAll, daemon.Name)
	response := &keactrl.StatisticGetAllResponse{}

	// Forward commands to Kea.
	ctx := context.Background()

	var serialCmds []keactrl.SerializableCommand
	serialCmds = append(serialCmds, cmd)

	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, daemon, serialCmds, response)
	if err != nil {
		return err
	}

	if err := cmdsResult.GetFirstError(); err != nil {
		return err
	}

	err = response.GetError()
	if err != nil {
		return errors.WithMessage(err, "the statistic-get-all command returned an error")
	}

	if response.Arguments == nil {
		return errors.Errorf("arguments missing in the statistic-get-all response")
	}

	// Due to historical reasons, Stork server expects the statistic of
	// declined leases for DHCPv6 will be named as "declined-nas" instead
	// of "declined-addresses". We cannot change the name in our structures
	// because code handling the global statistics expects the IPv4 and
	// IPv6 statistics to have unique names. So we need to rename the
	// statistic name in the response.
	if daemon.Name == daemonname.DHCPv6 {
		for _, sample := range response.Arguments {
			if sample.Name == "declined-addresses" {
				sample.Name = "declined-nas"
			}
		}
	}

	// Process the response.
	return statsPuller.processDaemonResponse(daemon, response)
}

// Processes a single daemon response.
func (statsPuller *StatsPuller) processDaemonResponse(daemon *dbmodel.Daemon, response *keactrl.StatisticGetAllResponse) error {
	// Lease statistic processing needs daemon's local subnets
	subnets, err := dbmodel.GetDaemonLocalSubnets(statsPuller.DB, daemon.ID)
	if err != nil {
		return err
	}

	// Prepare a map that will speed up looking for LocalSubnet based on local
	// subnet ID.
	subnetsMap := make(map[int64]*dbmodel.LocalSubnet)
	for _, sn := range subnets {
		subnetsMap[sn.LocalSubnetID] = sn
	}

	var lastErr error
	err = statsPuller.storeDaemonStats(response, subnetsMap, daemon)
	if err != nil {
		log.WithError(err).Error("Error handling subnet statistics  in " +
			"the statistic-get-all response")
		lastErr = err
	}

	err = statsPuller.Response4Handler(daemon, response)
	if err != nil {
		log.WithError(err).Error("Error handling RPS DHCPv4 statistics " +
			" in the statistic-get-all response")
		lastErr = err
	}

	err = statsPuller.Response6Handler(daemon, response)
	if err != nil {
		log.WithError(err).Error("Error handling RPS DHCPv6 statistics " +
			" in the statistic-get-all response")
		lastErr = err
	}

	return lastErr
}
