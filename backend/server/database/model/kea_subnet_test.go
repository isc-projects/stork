package dbmodel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
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

	// Add Kea daemon with DHCPv4 subnets in two shared networks.
	accessPoints := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1114,
			Key:     "",
		},
	}
	d4 := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
	configRaw := &map[string]interface{}{
		"Dhcp4": map[string]interface{}{
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
	}
	configJSON, _ := json.Marshal(configRaw)
	err = d4.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d4)
	require.NoError(t, err)
	// Specify the shared networks to be committed as global shared networks
	// and associated with this daemon.
	daemonNetworks := []SharedNetwork{
		{
			Name:   "frog",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.1.0.0/24",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d4.ID,
							LocalSubnetID: 11,
							AddressPools: []AddressPool{
								{
									LowerBound: "192.1.0.1",
									UpperBound: "192.1.0.100",
								},
								{
									LowerBound: "192.1.0.150",
									UpperBound: "192.1.0.200",
								},
							},
						},
					},
				},
			},
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID: d4.ID,
				},
			},
		},
		{
			Name:   "mouse",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.2.0.0/24",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d4.ID,
							LocalSubnetID: 12,
							AddressPools: []AddressPool{
								{
									LowerBound: "192.2.0.1",
									UpperBound: "192.2.0.100",
								},
								{
									LowerBound: "192.2.0.150",
									UpperBound: "192.2.0.200",
								},
							},
						},
					},
				},
			},
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID: d4.ID,
				},
			},
		},
	}

	daemonSubnets := []Subnet{
		{
			Prefix: "192.168.0.0/24",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d4.ID,
					LocalSubnetID: 1,
					AddressPools: []AddressPool{
						{
							LowerBound: "192.168.0.1",
							UpperBound: "192.168.0.100",
						},
						{
							LowerBound: "192.168.0.150",
							UpperBound: "192.168.0.200",
						},
					},
				},
			},
		},
	}

	_, err = CommitNetworksIntoDB(db, daemonNetworks, daemonSubnets)
	require.NoError(t, err)

	// Add Kea daemon with DHCPv6 subnets, one global and one within a shared network.
	accessPoints6 := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1116,
			Key:     "",
		},
	}
	d6 := NewDaemon(m, daemonname.DHCPv6, true, accessPoints6)
	configRaw = &map[string]interface{}{
		"Dhcp6": map[string]interface{}{
			"subnet6": []map[string]interface{}{{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			}},
			"shared-networks": []map[string]interface{}{{
				"name": "fox",
				"subnet6": []map[string]interface{}{{
					"id":     21,
					"subnet": "5001:db8:1::/64",
				}},
			}},
		},
	}
	configJSON, _ = json.Marshal(configRaw)
	err = d6.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)
	err = AddDaemon(db, d6)
	require.NoError(t, err)

	daemonNetworks = []SharedNetwork{
		{
			Name:   "fox",
			Family: 6,
			Subnets: []Subnet{
				{
					Prefix: "5001:db8:1::/64",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d6.ID,
							LocalSubnetID: 21,
						},
					},
				},
			},
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DaemonID: d6.ID,
				},
			},
		},
	}

	daemonSubnets = []Subnet{
		{
			Prefix: "2001:db8:1::/64",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d6.ID,
					LocalSubnetID: 2,
				},
			},
		},
	}
	_, err = CommitNetworksIntoDB(db, daemonNetworks, daemonSubnets)
	require.NoError(t, err)

	// Kea daemon with DHCPv4 and DHCPv6 subnets.
	d46v4 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1146,
			Key:     "",
		},
	})
	configRaw = &map[string]interface{}{
		"Dhcp4": &map[string]interface{}{
			"subnet4": []map[string]interface{}{{
				"id":     3,
				"subnet": "192.118.0.0/24",
				"pools": []map[string]interface{}{{
					"pool": "192.118.0.1-192.118.0.200",
				}},
			}},
		},
	}
	configJSON, _ = json.Marshal(configRaw)
	err = d46v4.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d46v4)
	require.NoError(t, err)

	d46v6 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1147,
			Key:     "",
		},
	})
	configRaw = &map[string]interface{}{
		"Dhcp6": map[string]interface{}{
			"subnet6": []map[string]interface{}{{
				"id":     4,
				"subnet": "3001:db8:1::/64",
				"pools": []map[string]interface{}{{
					"pool": "3001:db8:1::/80",
				}},
				"pd-pools": []map[string]interface{}{
					{
						"prefix":        "3001:db8:1:1::",
						"prefix-len":    80,
						"delegated-len": 96,
					},
					{
						"prefix":              "3001:db8:1:2::",
						"prefix-len":          80,
						"delegated-len":       96,
						"excluded-prefix":     "3001:db8:1:2:1::",
						"excluded-prefix-len": 112,
					},
				},
			}},
		},
	}
	configJSON, _ = json.Marshal(configRaw)
	err = d46v6.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d46v6)
	require.NoError(t, err)

	daemonSubnets = []Subnet{
		{
			Prefix: "192.118.0.0/24",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d46v4.ID,
					LocalSubnetID: 3,
					AddressPools: []AddressPool{
						{
							LowerBound: "192.168.0.1",
							UpperBound: "192.168.0.200",
						},
					},
				},
			},
		},
		{
			Prefix: "3001:db8:1::/64",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d46v6.ID,
					LocalSubnetID: 4,
					AddressPools: []AddressPool{
						{
							LowerBound: "3001:db8:1::",
							UpperBound: "3001:db8:1:0:ffff::ffff",
						},
					},
					PrefixPools: []PrefixPool{
						{
							Prefix:       "3001:db8:1:1::/80",
							DelegatedLen: 96,
						},
						{
							Prefix:         "3001:db8:1:2::/80",
							DelegatedLen:   96,
							ExcludedPrefix: "3001:db8:1:2:1::/112",
						},
					},
				},
			},
		},
	}

	_, err = CommitNetworksIntoDB(db, []SharedNetwork{}, []Subnet{daemonSubnets[0]})
	require.NoError(t, err)
	_, err = CommitNetworksIntoDB(db, []SharedNetwork{}, []Subnet{daemonSubnets[1]})
	require.NoError(t, err)

	// Get all subnets.
	subnets, total, err := GetSubnetsByPage(db, 0, 10, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)

	localSubnetIDs := []int64{}
	for _, sn := range subnets {
		require.Len(t, sn.LocalSubnets, 1)
		localSubnetIDs = append(localSubnetIDs, sn.LocalSubnets[0].LocalSubnetID)
		switch sn.LocalSubnets[0].LocalSubnetID {
		case 1:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 2)
		case 2:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 0)
		case 7:
			require.Len(t, sn.LocalSubnets[0].PrefixPools, 2)
			require.Len(t, sn.LocalSubnets[0].AddressPools, 1)
		case 11:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 2)
		case 12:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 2)
		case 21:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 0)
		default:
			require.Len(t, sn.LocalSubnets[0].AddressPools, 1)
		}
	}
	// Make sure that all subnets have local subnet ids set.
	require.ElementsMatch(t, localSubnetIDs, []int64{1, 2, 3, 4, 11, 12, 21})

	// Get subnets from daemon d4
	subnets, total, err = GetSubnetsByPage(db, 0, 10, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)

	// Verify daemon associations are correct
	subnetsByDaemon := make(map[int64][]Subnet)
	for _, s := range subnets {
		require.Len(t, s.LocalSubnets, 1)
		daemonID := s.LocalSubnets[0].DaemonID
		subnetsByDaemon[daemonID] = append(subnetsByDaemon[daemonID], s)
	}

	// d4 should have 3 subnets
	require.Len(t, subnetsByDaemon[d4.ID], 3)
	// Verify local subnet IDs for d4
	d4LocalIDs := []int64{}
	for _, s := range subnetsByDaemon[d4.ID] {
		d4LocalIDs = append(d4LocalIDs, s.LocalSubnets[0].LocalSubnetID)
	}
	require.ElementsMatch(t, []int64{1, 11, 12}, d4LocalIDs)

	// Get subnets from the combined daemons (d46v4 and d46v6)
	// Verify d46v4 and d46v6 each have their subnets
	require.Len(t, subnetsByDaemon[d46v4.ID], 1)
	require.Len(t, subnetsByDaemon[d46v6.ID], 1)
	require.EqualValues(t, 3, subnetsByDaemon[d46v4.ID][0].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, 4, subnetsByDaemon[d46v6.ID][0].LocalSubnets[0].LocalSubnetID)

	// Get IPv4 subnets
	filters := &SubnetsByPageFilters{
		Family: newPtr(int64(4)),
	}
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, subnets, 4)
	for _, s := range subnets {
		require.Len(t, s.LocalSubnets, 1)
	}
	for _, s := range subnets {
		require.Contains(t, []int64{1, 3, 11, 12}, s.LocalSubnets[0].LocalSubnetID)
	}

	// Get IPv6 subnets
	filters.Family = newPtr(int64(6))
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, subnets, 3)
	for _, s := range subnets {
		require.Len(t, s.LocalSubnets, 1)
	}
	for _, s := range subnets {
		require.Contains(t, []int64{2, 4, 21}, s.LocalSubnets[0].LocalSubnetID)
	}

	// Get subnets by text '118.0.0/2'
	filters = &SubnetsByPageFilters{
		Text: newPtr("118.0.0/2"),
	}
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d46v4.ID, subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, 3, subnets[0].LocalSubnets[0].LocalSubnetID)

	// get subnets by text '0.150-192.168'
	filters.Text = newPtr("0.150-192.168")
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d4.ID, subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, 1, subnets[0].LocalSubnets[0].LocalSubnetID)

	// get subnets by text '118' - should find the d46v4 daemon subnet
	filters.Text = newPtr("118")
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d46v4.ID, subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, 3, subnets[0].LocalSubnets[0].LocalSubnetID)

	// get v4 subnets by text '200'
	filters.Family = newPtr(int64(4))
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d46v4.ID, subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, 3, subnets[0].LocalSubnets[0].LocalSubnetID)

	// get v4 subnets by local subnet ID '2'
	filters = &SubnetsByPageFilters{
		LocalSubnetID: newPtr(int64(2)),
	}
	subnets, total, err = GetSubnetsByPage(db, 0, 10, filters, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, subnets, 1)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d6.ID, subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, 2, subnets[0].LocalSubnets[0].LocalSubnetID)

	// get subnets sorted by id ascending
	subnets, total, err = GetSubnetsByPage(db, 0, 10, nil, "", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)
	require.EqualValues(t, 1, subnets[0].ID)
	require.EqualValues(t, 7, subnets[6].ID)

	// get subnets sorted by id descending
	subnets, total, err = GetSubnetsByPage(db, 0, 10, nil, "", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)
	require.EqualValues(t, 7, subnets[0].ID)
	require.EqualValues(t, 1, subnets[6].ID)

	// get subnets sorted by prefix ascending
	subnets, total, err = GetSubnetsByPage(db, 0, 10, nil, "prefix", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)
	require.EqualValues(t, 1, subnets[0].ID)
	require.EqualValues(t, 4, subnets[6].ID)

	// get subnets sorted by prefix descending
	subnets, total, err = GetSubnetsByPage(db, 0, 10, nil, "prefix", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, subnets, 7)
	require.EqualValues(t, 4, subnets[0].ID)
	require.EqualValues(t, 1, subnets[6].ID)
}

// Check if getting subnets works when there is no subnets in config of kea daemon.
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

	// Add Kea DHCPv4 daemon without subnets
	accessPoints := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1114,
			Key:     "",
		},
	}
	d4 := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
	configRaw := &map[string]interface{}{
		"Dhcp4": &map[string]interface{}{},
	}
	configJSON, _ := json.Marshal(configRaw)
	err = d4.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d4)
	require.NoError(t, err)

	// Get all subnets -> empty list should be returned
	subnets, total, err := GetSubnetsByPage(db, 0, 10, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Len(t, subnets, 0)
}

// Check that basic functionality of shared networks works, returns proper data and can be filtered.
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

	// add daemon with dhcp4 to machine
	d4 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1114,
			Key:     "",
		},
	})
	configRaw := &map[string]interface{}{
		"Dhcp4": map[string]interface{}{
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
	}
	configJSON, _ := json.Marshal(configRaw)
	err = d4.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d4)
	require.NoError(t, err)

	appNetworks := []SharedNetwork{
		{
			Name:   "frog",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.1.0.0/24",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d4.ID,
							LocalSubnetID: 11,
						},
					},
				},
			},
		},
		{
			Name:   "mouse",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.2.0.0/24",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d4.ID,
							LocalSubnetID: 12,
						},
					},
				},
				{
					Prefix: "192.3.0.0/24",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d4.ID,
							LocalSubnetID: 13,
						},
					},
				},
			},
		},
	}

	appSubnets := []Subnet{
		{
			Prefix: "192.168.0.0/24",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d4.ID,
					LocalSubnetID: 1,
				},
			},
		},
	}
	_, err = CommitNetworksIntoDB(db, appNetworks, appSubnets)
	require.NoError(t, err)

	// add daemon with dhcp6 to machine
	d6 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1116,
			Key:     "",
		},
	})
	configRaw = &map[string]interface{}{
		"Dhcp6": map[string]interface{}{
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
	}
	configJSON, _ = json.Marshal(configRaw)
	err = d6.SetKeaConfigFromJSON(configJSON)
	require.NoError(t, err)

	err = AddDaemon(db, d6)
	require.NoError(t, err)

	appNetworks = []SharedNetwork{
		{
			Name:   "fox",
			Family: 6,
			Subnets: []Subnet{
				{
					Prefix: "5001:db8:1::/64",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d6.ID,
							LocalSubnetID: 21,
						},
					},
				},
				{
					Prefix: "6001:db8:1::/64",
					LocalSubnets: []*LocalSubnet{
						{
							DaemonID:      d6.ID,
							LocalSubnetID: 22,
						},
					},
				},
			},
		},
	}

	appSubnets = []Subnet{
		{
			Prefix: "2001:db8:1::/64",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      d6.ID,
					LocalSubnetID: 2,
				},
			},
		},
	}
	_, err = CommitNetworksIntoDB(db, appNetworks, appSubnets)
	require.NoError(t, err)

	// Get all shared networks.
	networks, total, err := GetSharedNetworksByPage(db, 0, 10, 0, 0, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
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

	// Get shared networks for Kea daemon d4.
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, d4.ID, 0, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, networks, 2)

	require.Len(t, networks[0].Subnets, 1)
	require.Len(t, networks[0].Subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d4.ID, networks[0].Subnets[0].LocalSubnets[0].DaemonID)

	require.Len(t, networks[1].Subnets, 2)
	require.Len(t, networks[1].Subnets[0].LocalSubnets, 1)
	require.EqualValues(t, d4.ID, networks[1].Subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, d4.ID, networks[1].Subnets[1].LocalSubnets[0].DaemonID)

	require.ElementsMatch(t, []string{"frog", "mouse"}, []string{networks[0].Name, networks[1].Name})

	// Get shared networks for Kea daemon d6.
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 6, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, networks, 1)
	require.Len(t, networks[0].Subnets, 2)
	require.Len(t, networks[0].Subnets[0].LocalSubnets, 1)
	require.Len(t, networks[0].Subnets[1].LocalSubnets, 1)
	require.EqualValues(t, d6.ID, networks[0].Subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, d6.ID, networks[0].Subnets[1].LocalSubnets[0].DaemonID)
	require.Equal(t, "fox", networks[0].Name)

	// Get networks by text "mous".
	text := "mous"
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, &text, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, networks, 1)
	require.Len(t, networks[0].Subnets, 2)
	require.Len(t, networks[0].Subnets[0].LocalSubnets, 1)
	require.Len(t, networks[0].Subnets[1].LocalSubnets, 1)
	require.EqualValues(t, d4.ID, networks[0].Subnets[0].LocalSubnets[0].DaemonID)
	require.EqualValues(t, d4.ID, networks[0].Subnets[1].LocalSubnets[0].DaemonID)
	require.Equal(t, "mouse", networks[0].Name)

	// check sorting by id asc
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, nil, "", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, networks, 3)
	require.EqualValues(t, 1, networks[0].ID)
	require.EqualValues(t, 3, networks[2].ID)

	// check sorting by id desc
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, nil, "", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, networks, 3)
	require.EqualValues(t, 3, networks[0].ID)
	require.EqualValues(t, 1, networks[2].ID)

	// check sorting by name asc
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, nil, "name", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, networks, 3)
	require.EqualValues(t, "fox", networks[0].Name)
	require.EqualValues(t, "frog", networks[1].Name)
	require.EqualValues(t, "mouse", networks[2].Name)

	// check sorting by name desc
	networks, total, err = GetSharedNetworksByPage(db, 0, 10, 0, 0, nil, "name", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, networks, 3)
	require.EqualValues(t, "mouse", networks[0].Name)
	require.EqualValues(t, "frog", networks[1].Name)
	require.EqualValues(t, "fox", networks[2].Name)
}
