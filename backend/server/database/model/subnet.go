package dbmodel

import (
	"fmt"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	// log "github.com/sirupsen/logrus"
)

type Subnet struct {
	ID             int
	AppID          int
	Subnet         string
	Pools          []map[string]interface{}
	SharedNetwork  string
	MachineAddress string
	AgentPort      int64
}

type SharedNetwork struct {
	Name           string
	AppID          int
	Subnets        []map[string]interface{}
	MachineAddress string
	AgentPort      int64
}

// Fetches a collection of subnets from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned. Result can be filtered by appID (if it is different than 0)
// or by DHCP version: 4 or 6 (0 means no filtering).
func GetSubnetsByPage(db *pg.DB, offset int64, limit int64, appID int64, dhcpVer int64, text *string) ([]Subnet, int64, error) {
	var subnets []Subnet
	var err error

	if dhcpVer != 0 && dhcpVer != 4 && dhcpVer != 6 {
		return []Subnet{}, 0, fmt.Errorf("wrong DHCP version: %d", dhcpVer)
	}

	params := &struct {
		Appid  int64
		Offset int64
		Limit  int64
		Text   string
	}{}
	params.Appid = appID
	params.Offset = offset
	params.Limit = limit

	// Build a query do goes through apps and their configs and retrieves list of subnets
	// for both DHCPv4 and v6.
	// Example of such query:
	//
	// SELECT app_id, sn->>'id' as id, sn->>'subnet' as subnet, sn->'pools' as pools
	//   FROM ( SELECT id AS app_id, jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp4'->'subnet4') AS sn, '' as sharednetwork FROM app
	//          UNION
	//          SELECT id AS app_id, jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp6'->'subnet6') AS sn, '' as sharednetwork FROM app
	//          UNION
	//          SELECT id AS app_id,
	//                 jsonb_array_elements(jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp4'->'shared-networks')->'subnet4') AS sn,
	//                 jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp4'->'shared-networks')->>'name' AS sharednetwork
	//              FROM app
	//        ) sq
	//   WHERE sn->>'subnet' like '%192%' OR sn->>'pools' like '%192%' ORDER BY subnet, id, app_id OFFSET NULL LIMIT 10;
	//
	// It looks for v4 and v6 subnets with `192` in subnet or pools text.
	query := ``
	whereAppID := ` WHERE id = ?appid`

	sqSubnets := ` SELECT id AS app_id, machine_id, `
	sqSubnets += ` jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'subnet%d') AS sn, '' as shared_network`
	sqSubnets += ` FROM app`
	sqNetworkSubnets := ` SELECT id AS app_id, machine_id,`
	sqNetworkSubnets += ` jsonb_array_elements(jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'shared-networks')->'subnet%d') AS sn,`
	sqNetworkSubnets += ` jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'shared-networks')->>'name' AS shared_network`
	sqNetworkSubnets += ` FROM app`
	if dhcpVer == 0 {
		sq4 := fmt.Sprintf(sqSubnets, 4, 4)
		sq6 := fmt.Sprintf(sqSubnets, 6, 6)
		if appID != 0 {
			sq4 += whereAppID
			sq6 += whereAppID
		}
		query += sq4 + ` UNION ` + sq6

		query += ` UNION `

		sq4 = fmt.Sprintf(sqNetworkSubnets, 4, 4, 4)
		sq6 = fmt.Sprintf(sqNetworkSubnets, 6, 6, 6)
		if appID != 0 {
			sq4 += whereAppID
			sq6 += whereAppID
		}
		query += sq4 + ` UNION ` + sq6
	} else {
		query += fmt.Sprintf(sqSubnets, dhcpVer, dhcpVer)
		if appID != 0 {
			query += whereAppID
		}

		query += ` UNION `

		query += fmt.Sprintf(sqNetworkSubnets, dhcpVer, dhcpVer, dhcpVer)
		if appID != 0 {
			query += whereAppID
		}
	}

	query += `) sq`

	whereClause := ` `
	if text != nil {
		params.Text = "%" + *text + "%"
		whereClause = ` WHERE sn->>'subnet' like ?text`
		whereClause += ` OR sn->>'pools' like ?text`
		whereClause += ` OR shared_network like ?text`
	}

	// and then, first get total count
	queryTotal := `SELECT count(*) FROM (` + query + whereClause + `;`
	var total int64
	_, err = db.QueryOne(pg.Scan(&total), queryTotal, params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting subnets total")
	}

	// then retrieve given page of rows
	query2 := `SELECT app_id, sn->'id' as id, sn->>'subnet' as subnet, sn->'pools' as pools, shared_network,`
	query2 += `      machine.address as machine_address, machine.agent_port as agent_port FROM (` + query
	query2 += ` JOIN machine ON machine_id  = machine.id `
	query2 += whereClause
	query2 += ` ORDER BY subnet, id, app_id`
	query2 += ` OFFSET ?offset LIMIT ?limit;`

	_, err = db.Query(&subnets, query2, params)

	if err != nil {
		return []Subnet{}, 0, errors.Wrapf(err, "problem with getting subnets from db")
	}

	return subnets, total, nil
}

// Fetches a collection of shared networks from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned. Result can be filtered by appID (if it is different than 0)
// or by DHCP version: 4 or 6 (0 means no filtering).
func GetSharedNetworksByPage(db *pg.DB, offset int64, limit int64, appID int64, dhcpVer int64, text *string) ([]SharedNetwork, int64, error) {
	var sharedNetworks []SharedNetwork
	var err error

	if dhcpVer != 0 && dhcpVer != 4 && dhcpVer != 6 {
		return []SharedNetwork{}, 0, fmt.Errorf("wrong DHCP version: %d", dhcpVer)
	}

	params := &struct {
		Appid  int64
		Offset int64
		Limit  int64
		Text   string
	}{}
	params.Appid = appID
	params.Offset = offset
	params.Limit = limit

	// Build a query do goes through apps and their configs and retrieves list of shared networks with their subnets
	// for both DHCPv4 and v6.
	// Example of such query:
	//
	// SELECT app_id, name, subnets
	//    FROM (
	//       SELECT id as app_id, machine_id,
	//              jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp4'->'shared-networks')->>'name' AS name,
	//              jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp4'->'shared-networks')->'subnet4' AS subnets
	//          FROM app
	//       UNION
	//       SELECT id as app_id, machine_id,
	//              jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp6'->'shared-networks')->>'name' AS name,
	//              jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp6'->'shared-networks')->'subnet6' AS subnets
	//          FROM app
	//    ) sq
	// WHERE
	//    name like '%15.%'
	//    OR
	//    subnets::text like '%15.%'
	// ORDER BY name, app_id
	// OFFSET NULL LIMIT 10;

	query := ``
	whereAppID := ` WHERE id = ?appid`

	sq := ` SELECT id as app_id, machine_id,`
	sq += `        jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'shared-networks')->>'name' AS name,`
	sq += `        jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'shared-networks')->'subnet%d' AS subnets`
	sq += `        FROM app`
	if dhcpVer == 0 {
		sq4 := fmt.Sprintf(sq, 4, 4, 4)
		sq6 := fmt.Sprintf(sq, 6, 6, 6)
		if appID != 0 {
			sq4 += whereAppID
			sq6 += whereAppID
		}
		query += sq4 + ` UNION ` + sq6
	} else {
		query += fmt.Sprintf(sq, dhcpVer, dhcpVer, dhcpVer)
		if appID != 0 {
			query += whereAppID
		}
	}

	query += `) sq`

	whereClause := ` `
	if text != nil {
		params.Text = "%" + *text + "%"
		whereClause = ` WHERE name like ?text`
		whereClause += ` OR subnets::text like ?text`
	}

	// and then, first get total count
	queryTotal := `SELECT count(*) FROM (` + query + whereClause + `;`
	var total int64
	_, err = db.QueryOne(pg.Scan(&total), queryTotal, params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting shared networks total")
	}

	// then retrieve given page of rows
	query2 := `SELECT app_id, name, subnets, machine.address as machine_address, machine.agent_port as agent_port FROM (` + query
	query2 += ` JOIN machine ON machine_id  = machine.id `
	query2 += whereClause
	query2 += ` ORDER BY name, app_id`
	query2 += ` OFFSET ?offset LIMIT ?limit;`

	_, err = db.Query(&sharedNetworks, query2, params)


	if err != nil {
		return []SharedNetwork{}, 0, errors.Wrapf(err, "problem with getting shared networks from db")
	}

	return sharedNetworks, total, nil
}

// TODO: Questions:
// 1. how to identify subnet cross kea apps
// 2. how to identify shared network cross kea apps
