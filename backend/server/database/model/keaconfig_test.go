package dbmodel

import (
	"strings"
	"testing"

	"github.com/go-pg/pg/v10"
	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Test that KeaConfig isn't constructed from nil.
func TestNewKeaConfigFromNil(t *testing.T) {
	// Act
	configNil := NewKeaConfig(nil)

	// Assert
	require.Nil(t, configNil)
}

// Test that KeaConfig is constructed from an empty map.
func TestNewKeaConfigFromEmptyMap(t *testing.T) {
	// Act
	configEmpty := NewKeaConfig(&map[string]interface{}{})

	// Assert
	require.NotNil(t, configEmpty)
	require.NotNil(t, configEmpty.Raw)
	require.Nil(t, configEmpty.DHCPv4Config)
	require.Nil(t, configEmpty.DHCPv6Config)
	require.Nil(t, configEmpty.D2Config)
	require.Nil(t, configEmpty.CtrlAgentConfig)
}

// Test that KeaConfig is constructed from a filled map.
func TestNewKeaConfigFromFilledMap(t *testing.T) {
	// Act
	configFilled := NewKeaConfig(&map[string]any{"Dhcp4": map[string]any{"foo": "bar"}})

	// Assert
	require.NotNil(t, configFilled.DHCPv4Config)
}

// Verifies that the shared network instance can be created by parsing
// Kea configuration.
func TestNewSharedNetworkFromKea(t *testing.T) {
	network := &keaconfig.SharedNetwork6{
		Name: "foo",
		Subnet6: []keaconfig.Subnet6{
			{
				MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
					ID:     1,
					Subnet: "2001:db8:2::/64",
				},
				CommonSubnetParameters: keaconfig.CommonSubnetParameters{
					Reservations: []keaconfig.Reservation{
						{
							HWAddress: "01:02:03:04:05:06",
						},
					},
				},
			},
			{
				MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
					ID:     2,
					Subnet: "2001:db8:1::/64",
				},
			},
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	lookup := NewDHCPOptionDefinitionLookup()
	parsedNetwork, err := NewSharedNetworkFromKea(network, 6, daemon, HostDataSourceConfig, lookup)
	require.NoError(t, err)
	require.NotNil(t, parsedNetwork)
	require.Equal(t, "foo", parsedNetwork.Name)
	require.EqualValues(t, 6, parsedNetwork.Family)
	require.Len(t, parsedNetwork.Subnets, 2)

	require.Zero(t, parsedNetwork.Subnets[0].ID)
	require.Equal(t, "2001:db8:2::/64", parsedNetwork.Subnets[0].Prefix)
	require.Zero(t, parsedNetwork.Subnets[1].ID)
	require.Equal(t, "2001:db8:1::/64", parsedNetwork.Subnets[1].Prefix)

	require.Len(t, parsedNetwork.Subnets[0].Hosts, 1)
}

// Verifies that the subnet instance can be created by parsing Kea
// configuration.
func TestNewSubnetFromKea(t *testing.T) {
	keaSubnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			ID:     1,
			Subnet: "2001:db8:1::/64",
		},
		CommonSubnetParameters: keaconfig.CommonSubnetParameters{
			Pools: []keaconfig.Pool{
				{
					Pool: "2001:db8:1:1::/120",
				},
			},
			Reservations: []keaconfig.Reservation{
				{
					DUID: "01:02:03:04:05:06",
					IPAddresses: []string{
						"2001:db8:1::1",
						"2001:db8:1::2",
					},
					Prefixes: []string{
						"3000:1::/64",
						"3000:2::/64",
					},
				},
				{
					HWAddress: "01:01:01:01:01:01",
					IPAddresses: []string{
						"2001:db8:1::1",
						"2001:db8:1::2",
					},
					Prefixes: []string{
						"3000:1::/64",
						"3000:2::/64",
					},
				},
			},
			UserContext: map[string]any{
				"foo":         "bar",
				"subnet-name": "baz",
			},
		},
		PDPools: []keaconfig.PDPool{
			{
				Prefix:            "2001:db8:1:1::",
				PrefixLen:         96,
				DelegatedLen:      120,
				ExcludedPrefix:    "2001:db8:1:1:1::",
				ExcludedPrefixLen: 128,
			},
		},
	}

	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 234
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)
	require.NoError(t, err)
	require.NotNil(t, parsedSubnet)
	require.Zero(t, parsedSubnet.ID)
	require.Equal(t, "2001:db8:1::/64", parsedSubnet.Prefix)
	require.Len(t, parsedSubnet.LocalSubnets, 1)
	require.Len(t, parsedSubnet.LocalSubnets[0].AddressPools, 1)
	require.Equal(t, "2001:db8:1:1::", parsedSubnet.LocalSubnets[0].AddressPools[0].LowerBound)
	require.Equal(t, "2001:db8:1:1::ff", parsedSubnet.LocalSubnets[0].AddressPools[0].UpperBound)

	require.Len(t, parsedSubnet.LocalSubnets[0].PrefixPools, 1)
	require.Equal(t, "2001:db8:1:1::/96", parsedSubnet.LocalSubnets[0].PrefixPools[0].Prefix)
	require.EqualValues(t, 120, parsedSubnet.LocalSubnets[0].PrefixPools[0].DelegatedLen)
	require.Equal(t, "2001:db8:1:1:1::/128", parsedSubnet.LocalSubnets[0].PrefixPools[0].ExcludedPrefix)

	require.Len(t, parsedSubnet.Hosts, 2)
	require.Len(t, parsedSubnet.Hosts[0].HostIdentifiers, 1)
	require.Equal(t, "duid", parsedSubnet.Hosts[0].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, parsedSubnet.Hosts[0].HostIdentifiers[0].Value)
	require.Equal(t, "hw-address", parsedSubnet.Hosts[1].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 1, 1, 1, 1, 1}, parsedSubnet.Hosts[1].HostIdentifiers[0].Value)

	for i := 0; i < 2; i++ {
		require.Len(t, parsedSubnet.Hosts[i].LocalHosts, 1)
		require.Len(t, parsedSubnet.Hosts[0].LocalHosts[0].IPReservations, 4)
		require.Equal(t, "2001:db8:1::1", parsedSubnet.Hosts[1].LocalHosts[0].IPReservations[0].Address)
		require.Equal(t, "2001:db8:1::2", parsedSubnet.Hosts[1].LocalHosts[0].IPReservations[1].Address)
		require.Equal(t, "3000:1::/64", parsedSubnet.Hosts[1].LocalHosts[0].IPReservations[2].Address)
		require.Equal(t, "3000:2::/64", parsedSubnet.Hosts[1].LocalHosts[0].IPReservations[3].Address)
	}

	require.Equal(t, parsedSubnet.Hosts[0].ID, parsedSubnet.Hosts[0].LocalHosts[0].HostID)
	require.EqualValues(t, 234, parsedSubnet.Hosts[0].LocalHosts[0].DaemonID)
	require.Equal(t, HostDataSourceConfig, parsedSubnet.Hosts[0].LocalHosts[0].DataSource)

	require.Equal(t, "bar", parsedSubnet.LocalSubnets[0].UserContext["foo"])
	require.Equal(t, "baz", parsedSubnet.LocalSubnets[0].UserContext["subnet-name"])
}

// Test that the error is returned when the subnet prefix is invalid.
func TestNewSubnetFromKeaWithInvalidPrefix(t *testing.T) {
	// Arrange
	keaSubnet := keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "invalid",
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.Error(t, err)
	require.Nil(t, parsedSubnet)
}

// Test that the default mask is added to IPv4 subnet prefix if missing.
func TestNewSubnetFromKeaWithDefaultIPv4PrefixMask(t *testing.T) {
	// Arrange
	keaSubnet := keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "10.42.42.42",
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "10.42.42.42/32", parsedSubnet.Prefix)
}

// Test that the default mask is added to IPv6 subnet prefix if missing.
func TestNewSubnetFromKeaWithDefaultIPv6PrefixMask(t *testing.T) {
	// Arrange
	keaSubnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "fe80::42",
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "fe80::42/128", parsedSubnet.Prefix)
}

// Test that the IPv4 subnet prefix is converted from non-canonical to canonical form.
func TestNewSubnetFromKeaWithNonCanonicalIPv4Prefix(t *testing.T) {
	// Arrange
	keaSubnet := keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "10.42.42.42/8",
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "10.0.0.0/8", parsedSubnet.Prefix)
}

// Test that the IPv6 subnet prefix is converted from non-canonical to canonical form.
func TestNewSubnetFromKeaWithNonCanonicalIPv6Prefix(t *testing.T) {
	// Arrange
	keaSubnet := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "2001:db8:1::42/64",
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnetFromKea(&keaSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "2001:db8:1::/64", parsedSubnet.Prefix)
}

// Test that log targets can be created from parsed Kea logger config.
func TestNewLogTargetsFromKea(t *testing.T) {
	logger := keaconfig.Logger{
		Name: "logger-name",
		OutputOptions: []keaconfig.LoggerOutputOptions{
			{
				Output: "stdout",
			},
			{
				Output: "/tmp/log",
			},
		},
		Severity: "DEBUG",
	}

	targets := NewLogTargetsFromKea(logger)
	require.Len(t, targets, 2)
	require.Equal(t, "logger-name", targets[0].Name)
	require.Equal(t, "stdout", targets[0].Output)
	require.Equal(t, "debug", targets[0].Severity)
	require.Equal(t, "logger-name", targets[1].Name)
	require.Equal(t, "/tmp/log", targets[1].Output)
	require.Equal(t, "debug", targets[1].Severity)
}

// Test that appended value is properly scanned.
func TestKeaConfigAppendAndScanValue(t *testing.T) {
	// Arrange
	testCases := []struct {
		label string
		value map[string]interface{}
	}{
		{
			label: "nil",
			value: nil,
		},
		{
			label: "empty map",
			value: map[string]interface{}{},
		},
		{
			label: "flat map",
			value: map[string]interface{}{
				"foo": "foo",
				"bar": float64(42),
				"baz": true,
			},
		},
		{
			label: "nested map",
			value: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": map[string]interface{}{
						"baz": map[string]interface{}{},
					},
				},
			},
		},
	}

	for _, item := range testCases {
		testCase := item
		t.Run(testCase.label, func(t *testing.T) {
			inputConfig := NewKeaConfig(&testCase.value)
			var outputConfig KeaConfig
			// Act
			bytes, appendErr := inputConfig.AppendValue([]byte{}, 0)
			scanErr := outputConfig.ScanValue(
				storktest.NewPoolReaderMock(bytes, appendErr),
				len(bytes),
			)

			// Assert
			require.NoError(t, appendErr)
			require.NoError(t, scanErr)
			require.EqualValues(t, inputConfig.Config, outputConfig.Config)
		})
	}
}

// Test that KeaConfig and keaconfig.Config are parsed the same for NULL from database.
func TestKeaConfigIsAsKeaConfigMapForNullFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Config
	}

	var resConfig struct {
		Config *KeaConfig
	}

	query := "SELECT NULL::jsonb AS config"

	// Act
	_, errMap := db.Query(&resMap, query)
	_, errConfig := db.Query(&resConfig, query)

	// Assert
	require.NoError(t, errMap)
	require.NoError(t, errConfig)

	require.Nil(t, resMap.Config)
	require.Nil(t, resConfig.Config)
}

// Test that KeaConfig and keaconfig.Config are parsed the same for empty string from the database.
func TestKeaConfigIsAsKeaConfigMapForEmptyStringFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Config
	}

	var resConfig struct {
		Config *KeaConfig
	}

	query := "SELECT ''::jsonb AS config"

	// Act
	_, errMap := db.Query(&resMap, query)
	_, errConfig := db.Query(&resConfig, query)

	// Assert
	var pgErrMap pg.Error
	var pgErrConfig pg.Error

	require.ErrorAs(t, errMap, &pgErrMap)
	require.ErrorAs(t, errConfig, &pgErrConfig)

	// 22P02 - invalid input syntax for type json
	require.EqualValues(t, "22P02", pgErrMap.Field('C'))
	require.EqualValues(t, "22P02", pgErrConfig.Field('C'))
}

// Test that KeaConfig and keaconfig.Config are parsed the same for empty JSON from the database.
func TestKeaConfigIsAsKeaConfigMapForEmptyJSONFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Config
	}

	var resConfig struct {
		Config *KeaConfig
	}

	query := "SELECT '{}'::jsonb AS config"

	// Act
	_, errMap := db.Query(&resMap, query)
	_, errConfig := db.Query(&resConfig, query)

	// Assert
	require.NoError(t, errMap)
	require.NoError(t, errConfig)
	require.EqualValues(t, resMap.Config, resConfig.Config.Config)
}

// Test that KeaConfig and keaconfig.Config are parsed the same for JSON
// from a database containing single quote in value.
func TestKeaConfigIsAsKeaConfigMapForJSONWithSingleQuoteFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Config
	}

	var resConfig struct {
		Config *KeaConfig
	}

	query := `SELECT '{ "foo": "b''r" }'::jsonb AS config`

	// Act
	_, errMap := db.Query(&resMap, query)
	_, errConfig := db.Query(&resConfig, query)

	// Assert
	require.NoError(t, errMap)
	require.NoError(t, errConfig)
	require.EqualValues(t, resMap.Config, resConfig.Config.Config)
	rawConfig := resMap.Config.Raw
	require.EqualValues(t, "b'r", rawConfig["foo"])
}

// Test storing a huge Kea configuration in the database.
func TestStoreHugeKeaConfigInDatabase(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// 50MB
	hugeValue := strings.Repeat("a", 50*1024*1024)
	rawConfig := map[string]any{"Dhcp4": map[string]any{"b": hugeValue}}
	keaConfig := NewKeaConfig(&rawConfig)

	machine := &Machine{
		Address:   "localhost",
		AgentPort: 3000,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := NewKeaDaemon("dhcp4", true)
	err = daemon.SetConfig(keaConfig)
	require.NoError(t, err)

	addedDaemons, err := AddApp(db, &App{
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons:   []*Daemon{daemon},
	})
	require.NoError(t, err)

	addedDaemonID := addedDaemons[0].ID
	addedDaemon, err := GetDaemonByID(db, addedDaemonID)
	require.NoError(t, err)
	require.EqualValues(t, keaConfig.Config, addedDaemon.KeaDaemon.Config.Config)
}

// Test that nil value is stored as a database NULL (not JSON null) in the database.
func TestStoreNilValueInDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "localhost",
		AgentPort: 3000,
	}
	_ = AddMachine(db, machine)
	daemon := NewKeaDaemon("dhcp4", true)
	daemon.KeaDaemon.Config = nil

	_, _ = AddApp(db, &App{
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons:   []*Daemon{daemon},
	})

	// Act
	dbDaemon, err := GetDaemonByID(db, daemon.ID)

	// Assert
	require.NoError(t, err)
	require.Nil(t, dbDaemon.KeaDaemon.Config)
}

// Test creating a host from a DHCPv4 reservation.
func TestNewHostFromKeaDHCPv4Reservation(t *testing.T) {
	reservation := keaconfig.Reservation{
		HWAddress:      "01:02:03:04:05:06",
		IPAddress:      "192.0.2.1",
		Hostname:       "foo.example.org",
		ClientClasses:  []string{"foo", "bar"},
		NextServer:     "192.0.2.2",
		BootFileName:   "/tmp/boot",
		ServerHostname: "host.example.org",
		OptionData: []keaconfig.SingleOptionData{
			{
				AlwaysSend: true,
				Code:       5,
				CSVFormat:  true,
				Data:       "10.0.1.1",
				Name:       "domain-name-server",
				Space:      dhcpmodel.DHCPv4OptionSpace,
			},
		},
	}
	daemon := &Daemon{
		ID: 1,
	}
	lookup := NewDHCPOptionDefinitionLookup()
	host, err := NewHostFromKeaConfigReservation(reservation, daemon, HostDataSourceAPI, lookup)
	require.NoError(t, err)
	require.NotNil(t, host)

	require.Len(t, host.HostIdentifiers, 1)
	require.Equal(t, "hw-address", host.HostIdentifiers[0].Type)
	require.Equal(t, []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}, host.HostIdentifiers[0].Value)
	require.Len(t, host.LocalHosts, 1)
	require.Equal(t, "foo.example.org", host.LocalHosts[0].Hostname)
	require.Len(t, host.LocalHosts[0].IPReservations, 1)
	require.Equal(t, "192.0.2.1", host.LocalHosts[0].IPReservations[0].Address)
	require.EqualValues(t, 1, host.LocalHosts[0].DaemonID)
	require.Equal(t, "/tmp/boot", host.LocalHosts[0].BootFileName)
	require.Len(t, host.LocalHosts[0].ClientClasses, 2)
	require.Equal(t, "foo", host.LocalHosts[0].ClientClasses[0])
	require.Equal(t, "bar", host.LocalHosts[0].ClientClasses[1])
	require.Len(t, host.LocalHosts[0].DHCPOptionSet.Options, 1)
	require.Equal(t, HostDataSourceAPI, host.LocalHosts[0].DataSource)
	require.Equal(t, "192.0.2.2", host.LocalHosts[0].NextServer)
	require.Equal(t, "host.example.org", host.LocalHosts[0].ServerHostname)
}

// Test creating a host from a DHCPv6 reservation.
func TestNewHostFromKeaDHCPv6Reservation(t *testing.T) {
	reservation := keaconfig.Reservation{
		DUID:        "01:02:03:04:05:06",
		IPAddresses: []string{"2001:db8:1::1", "2001:db8:1::2"},
		Prefixes:    []string{"3000::/116"},
	}
	daemon := &Daemon{
		ID: 1,
	}
	lookup := NewDHCPOptionDefinitionLookup()
	host, err := NewHostFromKeaConfigReservation(reservation, daemon, HostDataSourceAPI, lookup)
	require.NoError(t, err)
	require.NotNil(t, host)

	require.Len(t, host.HostIdentifiers, 1)
	require.Equal(t, "duid", host.HostIdentifiers[0].Type)
	require.Equal(t, []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}, host.HostIdentifiers[0].Value)
	require.Len(t, host.LocalHosts, 1)
	require.Len(t, host.LocalHosts[0].IPReservations, 3)
	require.Equal(t, "2001:db8:1::1", host.LocalHosts[0].IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", host.LocalHosts[0].IPReservations[1].Address)
	require.Equal(t, "3000::/116", host.LocalHosts[0].IPReservations[2].Address)
	require.EqualValues(t, 1, host.LocalHosts[0].DaemonID)
	require.Equal(t, HostDataSourceAPI, host.LocalHosts[0].DataSource)
}
