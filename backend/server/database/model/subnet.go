package dbmodel

import (
	"fmt"

	"github.com/go-pg/pg/v9"
)

type Subnet struct {
	ID     string
	AppID  int
	Subnet string
	Pools  []map[string]interface{}
}

// Fetches a collection of subnets from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned. Result can be filtered by appID (if it is different than 0)
// or by DHCP version: 4 or 6 (0 means no filtering).
func GetSubnetsByPage(db *pg.DB, offset int64, limit int64, appID int64, dhcpVer int64, text *string) ([]Subnet, error) {
	var subnets []Subnet
	var err error

	if dhcpVer != 0 && dhcpVer != 4 && dhcpVer != 6 {
		return []Subnet{}, fmt.Errorf("wrong DHCP version: %d", dhcpVer)
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

	query := `SELECT app_id, sn->>'id' as id, sn->>'subnet' as subnet, sn->'pools' as pools FROM (`
	whereAppID := ` WHERE id = ?appid`

	sq := ` SELECT id AS app_id, jsonb_array_elements(jsonb_array_elements(details->'Daemons')->'Config'->'Dhcp%d'->'subnet%d') AS sn FROM app`
	if dhcpVer == 0 {
		sq4 := fmt.Sprintf(sq, 4, 4)
		sq6 := fmt.Sprintf(sq, 6, 6)
		if appID != 0 {
			sq4 += whereAppID
			sq6 += whereAppID
		}
		query += sq4 + ` UNION ` + sq6
	} else {
		query += fmt.Sprintf(sq, dhcpVer, dhcpVer)
		if appID != 0 {
			query += whereAppID
		}
	}

	query += `) sq`

	if text != nil {
		params.Text = "%" + *text + "%"
		query += ` WHERE sn->>'subnet' like ?text`
		query += ` OR sn->>'pools' like ?text`
	}

	query += ` ORDER BY subnet, id, app_id`
	query += ` OFFSET ?offset LIMIT ?limit;`

	_, err = db.Query(&subnets, query, params)

	if err != nil {
		return []Subnet{}, err
	}

	return subnets, nil
}
