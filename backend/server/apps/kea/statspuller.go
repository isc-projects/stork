package kea

import (
	"context"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

type StatsPuller struct {
	*agentcomm.PeriodicPuller
}

// Create a StatsPuller object that in background pulls Kea stats about leases.
// Beneath it spawns a goroutine that pulls stats periodically from Kea apps (that are stored in database).
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents) (*StatsPuller, error) {
	statsPuller := &StatsPuller{}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Stats", "kea_stats_puller_interval",
		statsPuller.pullLeaseStats)
	if err != nil {
		return nil, err
	}
	statsPuller.PeriodicPuller = periodicPuller
	return statsPuller, nil
}

// Shutdown StatsPuller. It stops goroutine that pulls stats.
func (statsPuller *StatsPuller) Shutdown() {
	statsPuller.PeriodicPuller.Shutdown()
}

// Pull stats periodically for all Kea apps which Stork is monitoring. The function returns a number
// of apps for which the stats were successfully pulled and last encountered error.
func (statsPuller *StatsPuller) pullLeaseStats() (int, error) {
	// get list of all kea apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.Db, dbmodel.AppTypeKea)
	if err != nil {
		return 0, err
	}

	// get lease stats from each kea app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getLeaseStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %+v: %+v", dbApp, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("completed pulling lease stats from Kea apps: %d/%d succeeded", appsOkCnt, len(dbApps))
	return appsOkCnt, lastErr
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
	agentcomm.KeaResponseHeader
	Arguments *StatLeaseGetArgs `json:"arguments,omitempty"`
}

// A key that is used in map that is mapping from (local subnet id, inet family) to LocalSubnet struct.
type localSubnetKey struct {
	LocalSubnetID int64
	Family        int
}

// Take a stats set from dhcp4 or dhcp6 daemon and store them in LocalSubnet in database.
func (statsPuller *StatsPuller) storeDaemonStats(resultSet *ResultSetInStatLeaseGet, subnetsMap map[localSubnetKey]*dbmodel.LocalSubnet, dbApp *dbmodel.App, family int) error {
	var lastErr error

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
				stats[name] = val
			}
		}
		if sn == nil {
			lastErr = errors.Errorf("cannot find LocalSubnet for app: %d, local subnet id: %d, family: %d", dbApp.ID, lsnID, family)
			log.Error(lastErr.Error())
			continue
		}
		err := sn.UpdateStats(statsPuller.Db, stats)
		if err != nil {
			log.Errorf("problem with updating Kea stats for local subnet id %d, app id %d: %s", sn.LocalSubnetID, dbApp.ID, err.Error())
			lastErr = err
		}
	}
	return lastErr
}

// Get lease stats from given kea app.
func (statsPuller *StatsPuller) getLeaseStatsFromApp(dbApp *dbmodel.App) error {
	// prepare URL to CA
	ctrlPoint, err := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		return err
	}
	caURL := storkutil.HostWithPortURL(ctrlPoint.Address, ctrlPoint.Port)

	// get active dhcp daemons
	dhcpDaemons := make(agentcomm.KeaDaemons)
	found := false
	for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
		if d.Name == "dhcp4" || d.Name == "dhcp6" {
			dhcpDaemons[d.Name] = true
			found = true
		}
	}
	// if no dhcp daemons found then exit
	if !found {
		return nil
	}

	// issue 2 commands to dhcp daemons at once to get their lease stats for v4 and v6
	cmds := []*agentcomm.KeaCommand{}
	if dhcpDaemons["dhcp4"] {
		cmds = append(cmds, &agentcomm.KeaCommand{
			Command: "stat-lease4-get",
			Daemons: &dhcpDaemons,
		})
	}
	if dhcpDaemons["dhcp6"] {
		cmds = append(cmds, &agentcomm.KeaCommand{
			Command: "stat-lease6-get",
			Daemons: &dhcpDaemons,
		})
	}

	// forward commands to kea
	statsResp1 := []StatLeaseGetResponse{}
	statsResp2 := []StatLeaseGetResponse{}
	ctx := context.Background()
	cmdsResult, err := statsPuller.Agents.ForwardToKeaOverHTTP(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, caURL, cmds, &statsResp1, &statsResp2)
	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	// assign responses to v4 and v6 depending on active daemons
	var stats4Resp []StatLeaseGetResponse
	var stats6Resp []StatLeaseGetResponse
	if dhcpDaemons["dhcp4"] {
		stats4Resp = statsResp1
		if dhcpDaemons["dhcp6"] {
			stats6Resp = statsResp2
		}
	} else if dhcpDaemons["dhcp6"] {
		stats6Resp = statsResp1
	}

	// get app's local subnets
	subnets, err := dbmodel.GetAppLocalSubnets(statsPuller.Db, dbApp.ID)
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

	// process response from kea daemons
	var lastErr error
	for idx, srs := range [][]StatLeaseGetResponse{stats4Resp, stats6Resp} {
		family := 4
		if idx == 1 {
			family = 6
		}
		for _, sr := range srs {
			if sr.Arguments == nil {
				continue
			}
			err = statsPuller.storeDaemonStats(&sr.Arguments.ResultSet, subnetsMap, dbApp, family)
			if err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}
