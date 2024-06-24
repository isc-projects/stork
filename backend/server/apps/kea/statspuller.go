package kea

import (
	"context"
	"math"
	"math/big"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
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

// Part of response for stat-lease4-get and stat-lease6-get commands.
type ResultSetInStatLeaseGet struct {
	Columns []string
	Rows    [][]storkutil.BigIntJSON
}

// Part of response for stat-lease4-get and stat-lease6-get commands.
type StatLeaseGetArgs struct {
	ResultSet ResultSetInStatLeaseGet `json:"result-set"`
	Timestamp string
}

// Represents unmarshaled response from Kea daemon to stat-lease4-get and stat-lease6-get commands.
type StatLeaseGetResponse struct {
	keactrl.ResponseHeader
	Arguments *StatLeaseGetArgs `json:"arguments,omitempty"`
}

// A key that is used in map that is mapping from (local subnet id, inet family) to LocalSubnet struct.
type localSubnetKey struct {
	LocalSubnetID int64
	Family        int
}

// Process lease stats results from the given command response for given daemon.
func (statsPuller *StatsPuller) storeDaemonStats(response interface{}, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	var lastErr error
	var sr []StatLeaseGetResponse

	statsResp, ok := response.(*[]StatLeaseGetResponse)
	if !ok {
		return errors.Errorf("response is empty: %+v", sr)
	}

	sr = *statsResp
	if len(sr) == 0 {
		return errors.Errorf("response is empty: %+v", sr)
	}

	if sr[0].Arguments == nil {
		return errors.Errorf("missing arguments from Lease Stats response %+v", sr[0])
	}

	resultSet := &sr[0].Arguments.ResultSet
	if resultSet == nil {
		return errors.Errorf("missing ResultSet from Lease Stats response %+v", sr[0])
	}

	for _, row := range resultSet.Rows {
		stats := dbmodel.SubnetStats{}
		var sn *dbmodel.LocalSubnet
		var lsnID int64
		for colIdx, val := range row {
			name := resultSet.Columns[colIdx]
			if name == "subnet-id" {
				lsnID = val.BigInt().Int64()
				sn = subnetsMap[localSubnetKey{lsnID, family}]
			} else {
				// handle inconsistency in stats naming in different kea versions
				name = strings.Replace(name, "addreses", "addresses", 1)
				value := val.BigInt()
				if value.Sign() == -1 {
					// Handle negative statistics from older Kea versions.
					// Older Kea versions stored the statistics as uint64
					// but they were returned as int64.
					//
					// For the negative int64 values:
					// uint64 = maxUint64 + (int64 + 1)
					value = big.NewInt(0).Add(
						big.NewInt(0).SetUint64(math.MaxUint64),
						big.NewInt(0).Add(
							big.NewInt(1),
							value,
						),
					)
				}

				// Store the value as a best fit type to preserve compatibility
				// with the existing code. Some features expect the IPv4
				// statistics to be always stored as uint64, while IPv6 can be
				// uint64 or big int.
				stats.SetBigCounter(name, storkutil.NewBigCounterFromBigInt(value))
			}
		}
		if sn == nil {
			lastErr = errors.Errorf("cannot find LocalSubnet for app: %d, local subnet ID: %d, family: %d", dbApp.ID, lsnID, family)
			log.Error(lastErr.Error())
			continue
		}
		err := sn.UpdateStats(statsPuller.DB, stats)
		if err != nil {
			log.Errorf("Problem updating Kea stats for local subnet ID %d, app ID %d: %s", sn.LocalSubnetID, dbApp.ID, err.Error())
			lastErr = err
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
	responses := []interface{}{}

	// Iterate over active daemons, adding commands and response containers
	// for dhcp4 and dhcp6 daemons.
	for _, d := range dbApp.Daemons {
		if d.KeaDaemon != nil && d.Active {
			if d.KeaDaemon.Config != nil {
				// Ignore the daemons without the statistic hook to avoid
				// confusing error messages.
				if _, _, present := d.KeaDaemon.Config.GetHookLibrary("libdhcp_stat_cmds"); !present {
					continue
				}
			}
			switch d.Name {
			case dhcp4:
				// Add daemon, cmd, and response for DHCP4 lease stats
				cmdDaemons = append(cmdDaemons, d)
				dhcp4Daemons := []string{dhcp4}
				cmds = append(cmds, keactrl.NewCommandBase(keactrl.StatLease4Get, dhcp4Daemons...))

				responses = append(responses, &[]StatLeaseGetResponse{})

				// Add daemon, cmd and response for DHCP4 RPS stats if we have an RpsWorker
				if statsPuller.RpsWorker != nil {
					cmdDaemons = append(cmdDaemons, d)
					responses = append(responses, RpsAddCmd4(&cmds, dhcp4Daemons))
				}
			case dhcp6:

				// Add daemon, cmd and response for DHCP6 lease stats
				cmdDaemons = append(cmdDaemons, d)
				dhcp6Daemons := []string{dhcp6}
				cmds = append(cmds, keactrl.NewCommandBase(keactrl.StatLease6Get, dhcp6Daemons...))

				responses = append(responses, &[]StatLeaseGetResponse{})

				// Add daemon, cmd and response for DHCP6 RPS stats if we have an RpsWorker
				if statsPuller.RpsWorker != nil {
					cmdDaemons = append(cmdDaemons, d)
					responses = append(responses, RpsAddCmd6(&cmds, dhcp6Daemons))
				}
			}
		}
	}

	// If there are no commands, nothing to do
	if len(cmds) == 0 {
		return nil
	}

	// forward commands to kea
	ctx := context.Background()

	var serialCmds []keactrl.SerializableCommand
	for _, cmd := range cmds {
		serialCmds = append(serialCmds, cmd)
	}
	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp, serialCmds, responses...)
	if err != nil {
		return err
	}

	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	// Process the response for each command for each daemon.
	return statsPuller.processAppResponses(dbApp, cmds, cmdDaemons, responses)
}

// Iterates through the commands for each daemon and processes the command responses
// Was part of getStatsFromApp() until lint:backend complained about cognitive complexity.
func (statsPuller *StatsPuller) processAppResponses(dbApp *dbmodel.App, cmds []*keactrl.Command, cmdDaemons []*dbmodel.Daemon, responses []interface{}) error {
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
		switch cmdDaemons[idx].Name {
		case dhcp4:
			switch cmds[idx].Command {
			case keactrl.StatLease4Get:
				err = statsPuller.storeDaemonStats(responses[idx], subnetsMap, dbApp, 4)
				if err != nil {
					log.Errorf("Error handling stat-lease4-get response: %+v", err)
					lastErr = err
				}
			case keactrl.StatisticGet:
				err = statsPuller.RpsWorker.Response4Handler(cmdDaemons[idx], responses[idx])
				if err != nil {
					log.Errorf("Error handling statistic-get (v4) response: %+v", err)
					lastErr = err
				}
			default:
				// Impossible case.
			}

		case dhcp6:
			switch cmds[idx].Command {
			case keactrl.StatLease6Get:
				err = statsPuller.storeDaemonStats(responses[idx], subnetsMap, dbApp, 6)
				if err != nil {
					log.Errorf("Error handling stat-lease6-get response: %+v", err)
					lastErr = err
				}
			case keactrl.StatisticGet:
				err = statsPuller.RpsWorker.Response6Handler(cmdDaemons[idx], responses[idx])
				if err != nil {
					log.Errorf("Error handling statistic-get (v6) response: %+v", err)
					lastErr = err
				}
			default:
				// Impossible case.
			}
		}
	}

	return lastErr
}
