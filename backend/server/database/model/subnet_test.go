package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbtest "isc.org/stork/server/database/test"
)

// Check that basic functionality of subnets works, returns proper data and can be filtered.
func TestGetSubnetsByPageBasic(t *testing.T) {
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
							"id":     1,
							"subnet": "192.168.0.0/24",
							"pools": []map[string]interface{}{{
								"pool": "192.168.0.1-192.168.0.100",
							}, {
								"pool": "192.168.0.150-192.168.0.200",
							}},
						}},
						"shared-networks": []map[string]interface{}{{
							"name": "frog",
							"subnet4": []map[string]interface{}{{
								"id":     11,
								"subnet": "192.1.0.0/24",
								"pools": []map[string]interface{}{{
									"pool": "192.1.0.1-192.1.0.100",
								}, {
									"pool": "192.1.0.150-192.1.0.200",
								}},
							}},
						}, {
							"name": "mouse",
							"subnet4": []map[string]interface{}{{
								"id":     12,
								"subnet": "192.2.0.0/24",
								"pools": []map[string]interface{}{{
									"pool": "192.2.0.1-192.2.0.100",
								}, {
									"pool": "192.2.0.150-192.2.0.200",
								}},
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
							"id":     2,
							"subnet": "2001:db8:1::/64",
							"pools":  []map[string]interface{}{},
						}},
						"shared-networks": []map[string]interface{}{{
							"name": "fox",
							"subnet6": []map[string]interface{}{{
								"id":     21,
								"subnet": "5001:db8:1::/64",
								"pools":  []map[string]interface{}{},
							}},
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
							"id":     3,
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
							"id":     4,
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
	subnets, total, err := GetSubnetsByPage(db, 0, 10, 0, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(7), total)
	require.Len(t, subnets, 7)
	for _, sn := range subnets {
		switch sn.ID {
		case 1:
			require.Len(t, sn.Pools, 2)
		case 2:
			require.Len(t, sn.Pools, 0)
		case 11:
			require.Len(t, sn.Pools, 2)
		case 12:
			require.Len(t, sn.Pools, 2)
		case 21:
			require.Len(t, sn.Pools, 0)
		default:
			require.Len(t, sn.Pools, 1)
		}
	}

	// get subnets from app a4
	subnets, total, err = GetSubnetsByPage(db, 0, 10, a4.ID, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, subnets, 3)
	require.Equal(t, int(a4.ID), subnets[0].AppID)
	require.Equal(t, int(a4.ID), subnets[1].AppID)
	require.Equal(t, int(a4.ID), subnets[2].AppID)
	require.ElementsMatch(t, []int{1, 11, 12}, []int{subnets[0].ID, subnets[1].ID, subnets[2].ID})

	// get subnets from app a46
	subnets, total, err = GetSubnetsByPage(db, 0, 10, a46.ID, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, subnets, 2)
	require.True(t, (subnets[0].ID == 3 && subnets[1].ID == 4) ||
		(subnets[0].ID == 4 && subnets[1].ID == 3))

	// get v4 subnets
	subnets, total, err = GetSubnetsByPage(db, 0, 10, 0, 4, nil)
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	require.Len(t, subnets, 4)
	require.ElementsMatch(t, []int{1, 3, 11, 12}, []int{subnets[0].ID, subnets[1].ID, subnets[2].ID, subnets[3].ID})

	// get v6 subnets
	subnets, total, err = GetSubnetsByPage(db, 0, 10, 0, 6, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, subnets, 3)
	require.ElementsMatch(t, []int{2, 4, 21}, []int{subnets[0].ID, subnets[1].ID, subnets[2].ID})

	// get v4 subnets and app a4
	subnets, total, err = GetSubnetsByPage(db, 0, 10, a4.ID, 4, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, subnets, 3)
	require.ElementsMatch(t, []int{1, 11, 12}, []int{subnets[0].ID, subnets[1].ID, subnets[2].ID})

	// get subnets by text '118.0.0/2'
	text := "118.0.0/2"
	subnets, total, err = GetSubnetsByPage(db, 0, 10, 0, 0, &text)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, 3, subnets[0].ID)

	// get subnets by text '0.150-192.168'
	text = "0.150-192.168"
	subnets, total, err = GetSubnetsByPage(db, 0, 10, 0, 0, &text)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a4.ID), subnets[0].AppID)
	require.Equal(t, 1, subnets[0].ID)

	// get subnets by text '200' and app a46
	text = "200"
	subnets, total, err = GetSubnetsByPage(db, 0, 10, a46.ID, 0, &text)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, 3, subnets[0].ID)

	// get v4 subnets by text '200' and app a46
	text = "200"
	subnets, total, err = GetSubnetsByPage(db, 0, 10, a46.ID, 4, &text)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, subnets, 1)
	require.Equal(t, int(a46.ID), subnets[0].AppID)
	require.Equal(t, 3, subnets[0].ID)
}

// Check if getting subnets works when there is no subnets in config of kea app.
func TestGetSubnetsByPageNoSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add machine
	m := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// add app kea with dhcp4 to machine but with no subnets configured
	a4 := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      KeaAppType,
		CtrlPort:  1114,
		Active:    true,
		Details: AppKea{
			Daemons: []*KeaDaemon{{
				Config: &map[string]interface{}{
					"Dhcp4": &map[string]interface{}{},
				},
			}},
		},
	}
	err = AddApp(db, a4)
	require.NoError(t, err)

	// get all subnets -> empty list should be retruned
	subnets, total, err := GetSubnetsByPage(db, 0, 10, 0, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Len(t, subnets, 0)
}

// Check that basic functionality of shared newtorks works, returns proper data and can be filtered.
func TestGetSharedNetworksByPageBasic(t *testing.T) {
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
							"id":     1,
							"subnet": "192.168.0.0/24",
							"pools":  []map[string]interface{}{},
						}},
						"shared-networks": []map[string]interface{}{{
							"name": "frog",
							"subnet4": []map[string]interface{}{{
								"id":     11,
								"subnet": "192.1.0.0/24",
							}},
						}, {
							"name": "mouse",
							"subnet4": []map[string]interface{}{{
								"id":     12,
								"subnet": "192.2.0.0/24",
							}, {
								"id":     13,
								"subnet": "192.3.0.0/24",
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
							"id":     2,
							"subnet": "2001:db8:1::/64",
							"pools":  []map[string]interface{}{},
						}},
						"shared-networks": []map[string]interface{}{{
							"name": "fox",
							"subnet6": []map[string]interface{}{{
								"id":     21,
								"subnet": "5001:db8:1::/64",
							}, {
								"id":     22,
								"subnet": "6001:db8:1::/64",
							}},
						}},
					},
				},
			}},
		},
	}
	err = AddApp(db, a6)
	require.NoError(t, err)

	// get all shared networks
	networks, total, err := GetSharedNetworksByPage(db, 0, 10, 0, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, networks, 3)
	for _, net := range networks {
		require.Contains(t, []string{"frog", "mouse", "fox"}, net.Name)
		switch net.Name {
		case "frog":
			require.Len(t, net.Subnets, 1)
		case "mouse":
			require.Len(t, net.Subnets, 2)
		case "fox":
			require.Len(t, net.Subnets, 2)
		}
	}

	// get shared networks from app a4
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, a4.ID, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, networks, 2)
	require.Equal(t, int(a4.ID), networks[0].AppID)
	require.Equal(t, int(a4.ID), networks[1].AppID)
	require.ElementsMatch(t, []string{"frog", "mouse"}, []string{networks[0].Name, networks[1].Name})

	// get DHCPv6 networks
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 6, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, networks, 1)
	require.Equal(t, int(a6.ID), networks[0].AppID)
	require.Equal(t, "fox", networks[0].Name)

	// get networks by text "mous"
	text := "mous"
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, &text)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, networks, 1)
	require.Equal(t, int(a4.ID), networks[0].AppID)
	require.Equal(t, "mouse", networks[0].Name)
}
