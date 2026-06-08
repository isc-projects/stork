package kea

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Tests that config target key fields are extracted correctly.
func TestBuildConfigTargetKeyNoServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)

	// Act
	id, err := buildConfigTargetKey(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.EqualValues(t, 5432, id.DBPort)
}

// Tests that the server tag is included in the config target key.
func TestBuildConfigTargetKeyWithServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)

	// Act
	id, err := buildConfigTargetKey(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.EqualValues(t, 5432, id.DBPort)
}

// Tests that an error is returned when no config databases are configured.
func TestBuildConfigTargetKeyNoConfigDB(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		Name:      daemonname.DHCPv4,
		KeaDaemon: &dbmodel.KeaDaemon{},
	}

	err := daemon.SetKeaConfigFromJSON([]byte(`{ "Dhcp4": {} }`))
	require.NoError(t, err)

	// Act
	_, err = buildConfigTargetKey(daemon)

	// Assert
	require.ErrorContains(t, err, "no config databases configured")
}

// Tests that target iterator groups daemons sharing the same config target
// and calls the callback exactly once for that target. The daemons with the
// subnet_cmds hook are not grouped. The daemons with no hooks or both hooks
// are silently skipped.
func TestForEachUniqueConfigSource(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon1.ID = 1
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2.ID = 2
	daemon3 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server3"), keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds)
	daemon3.ID = 3
	daemon4 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server4"), keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds)
	daemon4.ID = 4
	daemon5 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server5"))
	daemon5.ID = 5
	daemon6 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server6"),
		keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds,
		keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds,
	)
	daemon6.ID = 6

	subnet := newTestSubnet(daemon1, daemon2, daemon3, daemon4, daemon5, daemon6)
	subnet.LocalSubnets[0].KeaParameters = &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("iterative")}
	subnet.LocalSubnets[1].KeaParameters = &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("random")}

	called := 0

	// Act
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(localSubnets []*dbmodel.LocalSubnet) error {
		called++
		switch localSubnets[0].DaemonID {
		case 1, 2:
			require.Len(t, localSubnets, 2)
			daemonIDs := []int64{localSubnets[0].DaemonID, localSubnets[1].DaemonID}
			require.ElementsMatch(t, []int64{1, 2}, daemonIDs)
		case 3:
			require.Len(t, localSubnets, 1)
			require.EqualValues(t, 3, localSubnets[0].DaemonID)
		case 4:
			require.Len(t, localSubnets, 1)
			require.EqualValues(t, 4, localSubnets[0].DaemonID)
		default:
			require.Fail(t, "unexpected daemon ID: %d", localSubnets[0].DaemonID)
		}
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 3, called)
}

// Tests that target iterator calls a function only once for each unique config
// backend database.
func TestForEachUniqueConsistentConfigSourceDeduplicatesCBCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		called++
		require.Equal(t, []string{"server1"}, serverTags)
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, called)
}

// Tests that the target iterator calls a function only once for two daemons
// sharing the same config backend, even if they have different server tags.
func TestForEachUniqueConsistentConfigSourceCollectsDistinctServerTags(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		called++
		require.ElementsMatch(t, []string{"server1", "server2"}, serverTags)
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, called)
}

// Tests that the target iterator processes all daemons with the subnet_cmds
// hook.
func TestForEachUniqueConsistentConfigSourceProcessesSubnetCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds)
	daemon1.ID = 1
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		called++
		require.Len(t, serverTags, 1)
		require.Equal(t, fmt.Sprintf("server%d", ls.DaemonID), serverTags[0])
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 2, called)
}

// Tests that the target iterator silently skips local subnets whose daemon
// or Kea configuration is nil.
func TestForEachUniqueConsistentConfigSourceSkipsNilConfig(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"))
	subnet := newTestSubnet(daemon)

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		require.Fail(t, "it should not been called")
		return nil
	})

	// Assert
	require.NoError(t, err)
}

// Tests that cb_cmds daemons sharing one config backend must use the same
// local subnet ID.
func TestForEachUniqueConsistentConfigSourceRejectsInconsistentLocalSubnetID(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 99,
		SubnetID:      subnet.ID,
	})

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		return nil
	})

	// Assert
	require.ErrorContains(t, err, "have inconsistent Local subnet ID")
}

// Tests that cb_cmds daemons sharing one config backend must use consistent
// local subnet data. If not, an error describing the first inconsistent field
// is returned.
func TestForEachUniqueConsistentConfigSourceRejectsInconsistentLocalSubnetData(t *testing.T) {
	testCases := []struct {
		configure         func(ls *dbmodel.LocalSubnet)
		inconsistentField string
	}{
		{
			configure: func(ls *dbmodel.LocalSubnet) {
				ls.KeaParameters = &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("iterative")}
			},
			inconsistentField: "Kea parameters",
		},
		{
			configure: func(ls *dbmodel.LocalSubnet) {
				ls.PrefixPools = []dbmodel.PrefixPool{{Prefix: "2001:db8:2::/64", DelegatedLen: 80}}
			},
			inconsistentField: "Prefix pools",
		},
		{
			configure: func(ls *dbmodel.LocalSubnet) {
				ls.AddressPools = []dbmodel.AddressPool{{LowerBound: "192.0.2.11", UpperBound: "192.0.2.20"}}
			},
			inconsistentField: "Address pools",
		},
		{
			configure: func(ls *dbmodel.LocalSubnet) {
				ls.UserContext = map[string]any{"site": "dc2"}
			},
			inconsistentField: "User context",
		},
		{
			configure: func(ls *dbmodel.LocalSubnet) {
				ls.LocalSubnetID = 24
			},
			inconsistentField: "Local subnet ID",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inconsistent %s", tc.inconsistentField), func(t *testing.T) {
			// Arrange
			daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
			daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
			daemon2.ID = 2

			subnet := newTestSubnet(daemon1)
			reference := subnet.LocalSubnets[0]
			reference.KeaParameters = &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("random")}
			reference.PrefixPools = []dbmodel.PrefixPool{{Prefix: "2001:db8:1::/64", DelegatedLen: 80}}
			reference.AddressPools = []dbmodel.AddressPool{{LowerBound: "192.0.2.10", UpperBound: "192.0.2.20"}}
			reference.UserContext = map[string]any{"site": "dc1"}

			localSubnet := &dbmodel.LocalSubnet{
				DaemonID:      daemon2.ID,
				Daemon:        daemon2,
				LocalSubnetID: reference.LocalSubnetID,
				SubnetID:      subnet.ID,
				KeaParameters: &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("random")},
				PrefixPools:   []dbmodel.PrefixPool{{Prefix: "2001:db8:1::/64", DelegatedLen: 80}},
				AddressPools:  []dbmodel.AddressPool{{LowerBound: "192.0.2.10", UpperBound: "192.0.2.20"}},
				UserContext:   map[string]any{"site": "dc1"},
			}
			tc.configure(localSubnet)
			subnet.LocalSubnets = append(subnet.LocalSubnets, localSubnet)

			// Act
			err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
				return nil
			})

			// Assert
			require.ErrorContains(t, err, fmt.Sprintf("have inconsistent %s", tc.inconsistentField))
		})
	}
}

// Test that cb_cmds daemons sharing one config backend with consistent local
// subnet data do not cause an error to be returned by the target iterator.
func TestForEachUniqueConsistentConfigSourceAcceptsConsistentLocalSubnetData(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	reference := subnet.LocalSubnets[0]
	reference.KeaParameters = &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("random")}
	reference.PrefixPools = []dbmodel.PrefixPool{{
		Prefix:       "2001:db8:1::/64",
		DelegatedLen: 80,
		KeaParameters: &keaconfig.PoolParameters{
			PoolID: 1,
		},
	}}
	reference.AddressPools = []dbmodel.AddressPool{
		{LowerBound: "192.0.2.10", UpperBound: "192.0.2.20"},
		{LowerBound: "2001:db8::10", UpperBound: "2001:db8::20"},
	}
	reference.UserContext = map[string]any{"site": "dc1"}

	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: reference.LocalSubnetID,
		SubnetID:      subnet.ID,
		KeaParameters: &keaconfig.SubnetParameters{Allocator: storkutil.Ptr("random")},
		PrefixPools: []dbmodel.PrefixPool{{
			Prefix:       "2001:db8:1::/64",
			DelegatedLen: 80,
			KeaParameters: &keaconfig.PoolParameters{
				PoolID: 1,
			},
		}},
		AddressPools: []dbmodel.AddressPool{
			{LowerBound: "192.0.2.10", UpperBound: "192.0.2.20"},
			{LowerBound: "2001:db8::10", UpperBound: "2001:db8::20"},
		},
		UserContext: map[string]any{"site": "dc1"},
	})

	// Act
	err := forEachUniqueConsistentConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		return nil
	})

	// Assert
	require.NoError(t, err)
}
