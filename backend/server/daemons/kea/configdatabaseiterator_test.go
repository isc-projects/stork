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

// Tests that config backend key fields are extracted correctly.
func TestBuildConfigBackendKeyNoServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, keaconfig.SubnetAlteringHookLibraryCBCmds)

	// Act
	id, err := buildConfigBackendKey(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.Equal(t, 5432, id.DBPort)
}

// Tests that the server tag is included in the config backend ID.
func TestBuildConfigBackendIDWithServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)

	// Act
	id, err := buildConfigBackendKey(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.Equal(t, 5432, id.DBPort)
}

// Tests that an error is returned when no config databases are configured.
func TestBuildConfigBackendKeyNoConfigDB(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		Name:      daemonname.DHCPv4,
		KeaDaemon: &dbmodel.KeaDaemon{},
	}

	err := daemon.SetKeaConfigFromJSON([]byte(`{ "Dhcp4": {} }`))
	require.NoError(t, err)

	// Act
	_, err = buildConfigBackendKey(daemon)

	// Assert
	require.ErrorContains(t, err, "no config databases configured")
}

// Tests that target iterator calls a function only once for each unique config
// backend database.
func TestForEachUniqueConfigSourceDeduplicatesCBCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
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
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
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
func TestForEachUniqueConfigSourceCollectsDistinctServerTags(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAlteringHookLibraryCBCmds)
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
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
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
func TestForEachUniqueConfigSourceProcessesSubnetCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibrarySubnetCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAlteringHookLibrarySubnetCmds)
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
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		called++
		require.Nil(t, serverTags)
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 2, called)
}

// Tests that the target iterator silently skips local subnets whose daemon
// or Kea configuration is nil.
func TestForEachUniqueConfigSourceSkipsNilConfig(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"))
	subnet := newTestSubnet(daemon)

	// Act
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		require.Fail(t, "it should not been called")
		return nil
	})

	// Assert
	require.NoError(t, err)
}

// Tests that cb_cmds daemons sharing one config backend must use the same
// local subnet ID.
func TestForEachUniqueConfigSourceRejectsInconsistentLocalSubnetID(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAlteringHookLibraryCBCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 99,
		SubnetID:      subnet.ID,
	})

	// Act
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		return nil
	})

	// Assert
	require.ErrorContains(t, err, "have inconsistent Local subnet ID")
}

// Tests that cb_cmds daemons sharing one config backend must use consistent
// local subnet data. If not, an error describing the first inconsistent field
// is returned.
func TestForEachUniqueConfigSourceRejectsInconsistentLocalSubnetData(t *testing.T) {
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
			daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
			daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAlteringHookLibraryCBCmds)
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
			err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
				return nil
			})

			// Assert
			require.ErrorContains(t, err, fmt.Sprintf("have inconsistent %s", tc.inconsistentField))
		})
	}
}

// Test that cb_cmds daemons sharing one config backend with consistent local
// subnet data do not cause an error to be returned by the target iterator.
func TestForEachUniqueConfigSourceAcceptsConsistentLocalSubnetData(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), keaconfig.SubnetAlteringHookLibraryCBCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server2"), keaconfig.SubnetAlteringHookLibraryCBCmds)
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
	err := forEachUniqueConfigSource(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet, serverTags []string) error {
		return nil
	})

	// Assert
	require.NoError(t, err)
}
