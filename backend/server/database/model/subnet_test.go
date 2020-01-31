package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbtest "isc.org/stork/server/database/test"
)

func TestGetSubnetsByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// add app kea with dhcp4 to machine
	a4 := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1114,
		Active:    true,
		Details: AppKea{
			Daemons: []*KeaDaemon{{
				Config: &map[string]interface{}{
					"Dhcp4": &map[string]interface{}{
						"subnet4": []map[string]interface{}{{
							"id":     "1",
							"subnet": "192.168.0.0/24",
							"pools": []map[string]interface{}{{
								"pool": "192.168.0.1-192.168.0.100",
							}, {
								"pool": "192.168.0.150-192.168.0.200",
							}},
						}},
					},
				},
			}},
		},
	}
	err = AddApp(db, a4)
	require.NoError(t, err)

	// add app kea with dhcp6 to machine
	a6 := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1116,
		Active:    true,
		Details: AppKea{
			Daemons: []*KeaDaemon{{
				Config: &map[string]interface{}{
					"Dhcp6": &map[string]interface{}{
						"subnet6": []map[string]interface{}{{
							"id":     "2",
							"subnet": "2001:db8:1::/64",
							"pools":  []map[string]interface{}{},
						}},
					},
				},
			}},
		},
	}
	err = AddApp(db, a6)
	require.NoError(t, err)

	// add app kea with dhcp4 and dhcp6 to machine
	a46 := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1146,
		Active:    true,
		Details: AppKea{
			Daemons: []*KeaDaemon{{
				Config: &map[string]interface{}{
					"Dhcp4": &map[string]interface{}{
						"subnet4": []map[string]interface{}{{
							"id":     "3",
							"subnet": "192.118.0.0/24",
							"pools": []map[string]interface{}{{
								"pool": "192.118.0.1-192.118.0.200",
							}},
						}},
					},
				},
			}, {
				Config: &map[string]interface{}{
					"Dhcp6": &map[string]interface{}{
						"subnet6": []map[string]interface{}{{
							"id":     "4",
							"subnet": "3001:db8:1::/64",
							"pools": []map[string]interface{}{{
								"pool": "3001:db8:1::/80",
							}},
						}},
					},
				},
			}},
		},
	}
	err = AddApp(db, a46)
	require.NoError(t, err)

	// get all subnets
	subnets, err := GetSubnetsByPage(db, 0, 10, 0, 0, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 4)
	for _, sn := range subnets {
		switch sn.ID {
		case "1":
			require.Len(t, sn.Pools, 2)
		case "2":
			require.Len(t, sn.Pools, 0)
		default:
			require.Len(t, sn.Pools, 1)
		}
	}

	// get subnets from app a4
	subnets, err = GetSubnetsByPage(db, 0, 10, a4.ID, 0, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a4.ID), subnets[0].AppID)
	require.Equal(t, "1", subnets[0].ID)

	// get subnets from app a46
	subnets, err = GetSubnetsByPage(db, 0, 10, a46.ID, 0, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
	require.True(t, (subnets[0].ID == "3" && subnets[1].ID == "4") ||
		(subnets[0].ID == "4" && subnets[1].ID == "3"))

	// get v4 subnets
	subnets, err = GetSubnetsByPage(db, 0, 10, 0, 4, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
	require.True(t, (subnets[0].ID == "1" && subnets[1].ID == "3") ||
		(subnets[0].ID == "3" && subnets[1].ID == "1"))

	// get v6 subnets
	subnets, err = GetSubnetsByPage(db, 0, 10, 0, 6, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 2)
	require.True(t, (subnets[0].ID == "2" && subnets[1].ID == "4") ||
		(subnets[0].ID == "4" && subnets[1].ID == "2"))

	// get v4 subnets and app a4
	subnets, err = GetSubnetsByPage(db, 0, 10, a4.ID, 4, nil)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, "1", subnets[0].ID)

	// get subnets by text '118.0.0/2'
	text := "118.0.0/2"
	subnets, err = GetSubnetsByPage(db, 0, 10, 0, 0, &text)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, "3", subnets[0].ID)

	// get subnets by text '0.150-192.168'
	text = "0.150-192.168"
	subnets, err = GetSubnetsByPage(db, 0, 10, 0, 0, &text)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a4.ID), subnets[0].AppID)
	require.Equal(t, "1", subnets[0].ID)

	// get subnets by text '200' and app a46
	text = "200"
	subnets, err = GetSubnetsByPage(db, 0, 10, a46.ID, 0, &text)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, "3", subnets[0].ID)

	// get v4 subnets by text '200' and app a46
	text = "200"
	subnets, err = GetSubnetsByPage(db, 0, 10, a46.ID, 4, &text)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, "3", subnets[0].ID)
}
