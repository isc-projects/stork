package kea

import (
	"context"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

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
			log.Errorf("error occurred while getting stats from app %d: %+v", dbApp.ID, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("completed pulling lease stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))

	// estimate addresses utilization for subnets
	subnets, err := dbmodel.GetSubnetsWithLocalSubnets(statsPuller.DB)
	if err != nil {
		return err
	}

	if len(subnets) == 0 {
		return lastErr
	}

	calculator := newUtilizationCalculator()

	// go through all Subnets and:
	// 1) estimate utilization per Subnet and per SharedNetwork
	// 2) estimate global stats
	for _, sn := range subnets {
		su := calculator.add(sn)
		err = sn.UpdateUtilization(
			statsPuller.DB,
			int16(1000*su.addressUtilization()),
			int16(1000*su.pdUtilization()),
		)

		if err != nil {
			lastErr = err
			log.Errorf("cannot update utilization (%.3f, %.3f) in subnet %d: %s",
				su.addressUtilization(), su.pdUtilization(), sn.ID, err)
			continue
		}
	}

	// shared network utilization
	for sharedNetworkID, u := range calculator.sharedNetworks {
		err = dbmodel.UpdateUtilizationInSharedNetwork(statsPuller.DB, sharedNetworkID,
			int16(1000*u.addressUtilization()),
			int16(1000*u.pdUtilization()))

		if err != nil {
			lastErr = err
			log.Errorf("cannot update utilization (%.3f, %.3f) in shared network %d: %s",
				u.addressUtilization(), u.pdUtilization(), sharedNetworkID, err)
			continue
		}
	}

	// global stats to collect
	statsMap := map[string]int64{
		"total-addresses":    calculator.global.totalAddresses,
		"assigned-addresses": calculator.global.totalAssignedAddresses,
		"declined-addresses": calculator.global.totalDeclinedAddresses,
		"total-nas":          calculator.global.totalNAs,
		"assigned-nas":       calculator.global.totalAssignedNAs,
		"declined-nas":       calculator.global.totalDeclinedNAs,
		"assigned-pds":       calculator.global.totalAssignedPDs,
		"total-pds":          calculator.global.totalPDs,
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
	Rows    [][]int
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
		return errors.Errorf("missing Arguments from Lease Stats response %+v", sr[0])
	}

	resultSet := &sr[0].Arguments.ResultSet
	if resultSet == nil {
		return errors.Errorf("missing ResultSet from Lease Stats response %+v", sr[0])
	}

	for _, row := range resultSet.Rows {
		stats := make(map[string]interface{})
		var sn *dbmodel.LocalSubnet
		var lsnID int64
		for colIdx, val := range row {
			name := resultSet.Columns[colIdx]
			if name == "subnet-id" {
				lsnID = int64(val)
				sn = subnetsMap[localSubnetKey{lsnID, family}]
			} else {
				// handle inconsistency in stats naming in different kea versions
				switch name {
				case "total-addreses":
					name = "total-addresses"
				case "assigned-addreses":
					name = "assigned-addresses"
				case "declined-addreses":
					name = "declined-addresses"
				default:
				}
				stats[name] = val
			}
		}
		if sn == nil {
			lastErr = errors.Errorf("cannot find LocalSubnet for app: %d, local subnet id: %d, family: %d", dbApp.ID, lsnID, family)
			log.Error(lastErr.Error())
			continue
		}
		err := sn.UpdateStats(statsPuller.DB, stats)
		if err != nil {
			log.Errorf("problem with updating Kea stats for local subnet id %d, app id %d: %s", sn.LocalSubnetID, dbApp.ID, err.Error())
			lastErr = err
		}
	}
	return lastErr
}

func (statsPuller *StatsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// get active dhcp daemons
	dhcpDaemons := make(keactrl.Daemons)
	found := false
	for _, d := range dbApp.Daemons {
		if d.KeaDaemon != nil && d.Active && (d.Name == "dhcp4" || d.Name == "dhcp6") {
			dhcpDaemons[d.Name] = true
			found = true
		}
	}
	// if no dhcp daemons found then exit
	if !found {
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
			switch d.Name {
			case dhcp4:
				// Add daemon, cmd, and response for DHCP4 lease stats
				cmdDaemons = append(cmdDaemons, d)
				dhcp4Daemons, _ := keactrl.NewDaemons(dhcp4)
				cmds = append(cmds, &keactrl.Command{
					Command: "stat-lease4-get",
					Daemons: dhcp4Daemons,
				})

				responses = append(responses, &[]StatLeaseGetResponse{})

				// Add daemon, cmd and response for DHCP4 RPS stats if we have an RpsWorker
				if statsPuller.RpsWorker != nil {
					cmdDaemons = append(cmdDaemons, d)
					responses = append(responses, RpsAddCmd4(&cmds, dhcp4Daemons))
				}
			case dhcp6:

				// Add daemon, cmd and response for DHCP6 lease stats
				cmdDaemons = append(cmdDaemons, d)
				dhcp6Daemons, _ := keactrl.NewDaemons(dhcp6)
				cmds = append(cmds, &keactrl.Command{
					Command: "stat-lease6-get",
					Daemons: dhcp6Daemons,
				})

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

	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp, cmds, responses...)
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
// Was part of getStatsFromApp() until lint_go complained about cognitive complexity.
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
			case "stat-lease4-get":
				err = statsPuller.storeDaemonStats(responses[idx], subnetsMap, dbApp, 4)
				if err != nil {
					log.Errorf("error handling stat-lease4-get response: %+v", err)
					lastErr = err
				}
			case "statistic-get":
				err = statsPuller.RpsWorker.Response4Handler(cmdDaemons[idx], responses[idx])
				if err != nil {
					log.Errorf("error handling statistic-get (v4) response: %+v", err)
					lastErr = err
				}
			}

		case dhcp6:
			switch cmds[idx].Command {
			case "stat-lease6-get":
				err = statsPuller.storeDaemonStats(responses[idx], subnetsMap, dbApp, 6)
				if err != nil {
					log.Errorf("error handling stat-lease6-get response: %+v", err)
					lastErr = err
				}
			case "statistic-get":
				err = statsPuller.RpsWorker.Response6Handler(cmdDaemons[idx], responses[idx])
				if err != nil {
					log.Errorf("error handling statistic-get (v6) response: %+v", err)
					lastErr = err
				}
			}
		}
	}

	return lastErr
}
