package dbmodel

import (
	"strings"
	"testing"

	"github.com/go-pg/pg/v10"
	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
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
	require.EqualValues(t, map[string]interface{}{}, *configEmpty.Map)
}

// Test that KeaConfig is constructed from a filled map.
func TestNewKeaConfigFromFilledMap(t *testing.T) {
	// Act
	configFilled := NewKeaConfig(&map[string]interface{}{"Dhcp4": "foo"})
	root, ok := configFilled.GetRootName()

	// Assert
	require.True(t, ok)
	require.EqualValues(t, "Dhcp4", root)
}

// Verifies that the shared network instance can be created by parsing
// Kea configuration.
func TestNewSharedNetworkFromKea(t *testing.T) {
	rawNetwork := map[string]interface{}{
		"name": "foo",
		"subnet6": []map[string]interface{}{
			{
				"id":     1,
				"subnet": "2001:db8:2::/64",
				"reservations": []interface{}{
					map[string]interface{}{
						"hw-address": "01:02:03:04:05:06",
					},
				},
			},
			{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			},
		},
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	lookup := NewDHCPOptionDefinitionLookup()
	parsedNetwork, err := NewSharedNetworkFromKea(&rawNetwork, 6, daemon, HostDataSourceConfig, lookup)
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
	rawSubnet := map[string]interface{}{
		"id":     1,
		"subnet": "2001:db8:1::/64",
		"pools": []interface{}{
			map[string]interface{}{
				"pool": "2001:db8:1:1::/120",
			},
		},
		"pd-pools": []interface{}{
			map[string]interface{}{
				"prefix":              "2001:db8:1:1::",
				"prefix-len":          96,
				"delegated-len":       120,
				"excluded-prefix":     "2001:db8:1:1:1::",
				"excluded-prefix-len": 128,
			},
		},
		"reservations": []interface{}{
			map[string]interface{}{
				"duid": "01:02:03:04:05:06",
				"ip-addresses": []interface{}{
					"2001:db8:1::1",
					"2001:db8:1::2",
				},
				"prefixes": []interface{}{
					"3000:1::/64",
					"3000:2::/64",
				},
			},
			map[string]interface{}{
				"hw-address": "01:01:01:01:01:01",
				"ip-addresses": []interface{}{
					"2001:db8:1::1",
					"2001:db8:1::2",
				},
				"prefixes": []interface{}{
					"3000:1::/64",
					"3000:2::/64",
				},
			},
		},
	}

	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 234
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet6FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)
	require.NoError(t, err)
	require.NotNil(t, parsedSubnet)
	require.Zero(t, parsedSubnet.ID)
	require.Equal(t, "2001:db8:1::/64", parsedSubnet.Prefix)
	require.Len(t, parsedSubnet.AddressPools, 1)
	require.Equal(t, "2001:db8:1:1::", parsedSubnet.AddressPools[0].LowerBound)
	require.Equal(t, "2001:db8:1:1::ff", parsedSubnet.AddressPools[0].UpperBound)

	require.Len(t, parsedSubnet.PrefixPools, 1)
	require.Equal(t, "2001:db8:1:1::/96", parsedSubnet.PrefixPools[0].Prefix)
	require.EqualValues(t, 120, parsedSubnet.PrefixPools[0].DelegatedLen)
	require.Equal(t, "2001:db8:1:1:1::/128", parsedSubnet.PrefixPools[0].ExcludedPrefix)

	require.Len(t, parsedSubnet.Hosts, 2)
	require.Len(t, parsedSubnet.Hosts[0].HostIdentifiers, 1)
	require.Equal(t, "duid", parsedSubnet.Hosts[0].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, parsedSubnet.Hosts[0].HostIdentifiers[0].Value)
	require.Equal(t, "hw-address", parsedSubnet.Hosts[1].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 1, 1, 1, 1, 1}, parsedSubnet.Hosts[1].HostIdentifiers[0].Value)

	for i := 0; i < 2; i++ {
		require.Len(t, parsedSubnet.Hosts[0].IPReservations, 4)
		require.Equal(t, "2001:db8:1::1", parsedSubnet.Hosts[1].IPReservations[0].Address)
		require.Equal(t, "2001:db8:1::2", parsedSubnet.Hosts[1].IPReservations[1].Address)
		require.Equal(t, "3000:1::/64", parsedSubnet.Hosts[1].IPReservations[2].Address)
		require.Equal(t, "3000:2::/64", parsedSubnet.Hosts[1].IPReservations[3].Address)
	}

	require.Len(t, parsedSubnet.Hosts[0].LocalHosts, 1)
	require.Equal(t, parsedSubnet.Hosts[0].ID, parsedSubnet.Hosts[0].LocalHosts[0].HostID)
	require.EqualValues(t, 234, parsedSubnet.Hosts[0].LocalHosts[0].DaemonID)
	require.Equal(t, HostDataSourceConfig, parsedSubnet.Hosts[0].LocalHosts[0].DataSource)
}

// Test that the error is returned when the subnet prefix is invalid.
func TestNewSubnetFromKeaWithInvalidPrefix(t *testing.T) {
	// Arrange
	rawSubnet := map[string]interface{}{
		"subnet": "invalid",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet4FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.Error(t, err)
	require.Nil(t, parsedSubnet)
}

// Test that the default mask is added to IPv4 subnet prefix if missing.
func TestNewSubnetFromKeaWithDefaultIPv4PrefixMask(t *testing.T) {
	// Arrange
	rawSubnet := map[string]interface{}{
		"subnet": "10.42.42.42",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet4FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "10.42.42.42/32", parsedSubnet.Prefix)
}

// Test that the default mask is added to IPv6 subnet prefix if missing.
func TestNewSubnetFromKeaWithDefaultIPv6PrefixMask(t *testing.T) {
	// Arrange
	rawSubnet := map[string]interface{}{
		"subnet": "fe80::42",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet6FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "fe80::42/128", parsedSubnet.Prefix)
}

// Test that the IPv4 subnet prefix is converted from non-canonical to canonical form.
func TestNewSubnetFromKeaWithNonCanonicalIPv4Prefix(t *testing.T) {
	// Arrange
	rawSubnet := map[string]interface{}{
		"subnet": "10.42.42.42/8",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet4FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "10.0.0.0/8", parsedSubnet.Prefix)
}

// Test that the IPv6 subnet prefix is converted from non-canonical to canonical form.
func TestNewSubnetFromKeaWithNonCanonicalIPv6Prefix(t *testing.T) {
	// Arrange
	rawSubnet := map[string]interface{}{
		"subnet": "2001:db8:1::42/64",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv6, true)
	daemon.ID = 42

	// Act
	lookup := NewDHCPOptionDefinitionLookup()
	parsedSubnet, err := NewSubnet6FromKea(&rawSubnet, daemon, HostDataSourceConfig, lookup)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "2001:db8:1::/64", parsedSubnet.Prefix)
}

// Verifies that the host instance can be created by parsing Kea
// DHCPv4 configuration.
func TestNewV4HostFromKea(t *testing.T) {
	rawHost := map[string]interface{}{
		"hw-address":      "01:02:03:04:05:06",
		"ip-address":      "192.0.2.5",
		"hostname":        "hostname.example.org",
		"next-server":     "192.0.2.3",
		"server-hostname": "my-server-host",
		"boot-file-name":  "/tmp/bootfile",
	}
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 123
	lookup := NewDHCPOptionDefinitionLookup()
	parsedHost, err := NewHostFromKea(&rawHost, daemon, HostDataSourceConfig, lookup)
	require.NoError(t, err)
	require.NotNil(t, parsedHost)

	// Host identifiers.
	require.Len(t, parsedHost.HostIdentifiers, 1)
	require.Equal(t, "hw-address", parsedHost.HostIdentifiers[0].Type)
	require.Len(t, parsedHost.IPReservations, 1)
	require.Equal(t, "192.0.2.5", parsedHost.IPReservations[0].Address)

	// Local host.
	require.Len(t, parsedHost.LocalHosts, 1)
	require.Equal(t, parsedHost.ID, parsedHost.LocalHosts[0].HostID)
	require.EqualValues(t, 123, parsedHost.LocalHosts[0].DaemonID)
	require.Equal(t, HostDataSourceConfig, parsedHost.LocalHosts[0].DataSource)

	// Boot fields.
	require.Equal(t, "192.0.2.3", parsedHost.LocalHosts[0].NextServer)
	require.Equal(t, "my-server-host", parsedHost.LocalHosts[0].ServerHostname)
	require.Equal(t, "/tmp/bootfile", parsedHost.LocalHosts[0].BootFileName)
}

// Verifies that the host instance can be created by parsing Kea
// DHCPv6 configuration.
func TestNewV6HostFromKea(t *testing.T) {
	rawHost := map[string]interface{}{
		"duid": "01:02:03:04",
		"ip-addresses": []interface{}{
			"2001:db8:1::1",
			"2001:db8:1::2",
		},
		"prefixes": []interface{}{
			"3000:1::/64",
			"3000:2::/64",
		},
		"hostname":       "hostname.example.org",
		"client-classes": []string{"foo", "bar"},
		"option-data": []interface{}{
			map[string]interface{}{
				"always-send": true,
				"code":        23,
				"csv-format":  true,
				"data":        "2001:db8:1::15,2001:db8:1::16",
				"name":        "dns-servers",
				"space":       "dhcp6",
			},
		},
	}

	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	daemon.ID = 123
	lookup := NewDHCPOptionDefinitionLookup()
	parsedHost, err := NewHostFromKea(&rawHost, daemon, HostDataSourceConfig, lookup)
	require.NoError(t, err)
	require.NotNil(t, parsedHost)

	// Host identifiers.
	require.Len(t, parsedHost.HostIdentifiers, 1)
	require.Equal(t, "duid", parsedHost.HostIdentifiers[0].Type)
	require.Len(t, parsedHost.IPReservations, 4)

	// IP reservations.
	require.Equal(t, "2001:db8:1::1", parsedHost.IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedHost.IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedHost.IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedHost.IPReservations[3].Address)
	require.Equal(t, "hostname.example.org", parsedHost.Hostname)
	require.Len(t, parsedHost.LocalHosts, 1)
	require.Equal(t, parsedHost.ID, parsedHost.LocalHosts[0].HostID)
	require.EqualValues(t, 123, parsedHost.LocalHosts[0].DaemonID)
	require.Equal(t, HostDataSourceConfig, parsedHost.LocalHosts[0].DataSource)

	// Client classes
	require.Len(t, parsedHost.LocalHosts[0].ClientClasses, 2)
	require.Equal(t, "foo", parsedHost.LocalHosts[0].ClientClasses[0])
	require.Equal(t, "bar", parsedHost.LocalHosts[0].ClientClasses[1])

	// DHCP options
	require.Len(t, parsedHost.LocalHosts[0].DHCPOptionSet, 1)
	require.True(t, parsedHost.LocalHosts[0].DHCPOptionSet[0].AlwaysSend)
	require.EqualValues(t, 23, parsedHost.LocalHosts[0].DHCPOptionSet[0].Code)
	require.Equal(t, "dns-servers", parsedHost.LocalHosts[0].DHCPOptionSet[0].Name)
	require.Equal(t, "dhcp6", parsedHost.LocalHosts[0].DHCPOptionSet[0].Space)
	require.Len(t, parsedHost.LocalHosts[0].DHCPOptionSet[0].Fields, 2)
	require.Equal(t, "ipv6-address", parsedHost.LocalHosts[0].DHCPOptionSet[0].Fields[0].FieldType)
	require.Len(t, parsedHost.LocalHosts[0].DHCPOptionSet[0].Fields[0].Values, 1)
	require.Equal(t, "2001:db8:1::15", parsedHost.LocalHosts[0].DHCPOptionSet[0].Fields[0].Values[0])

	// Make sure the hash is computed.
	require.NotEmpty(t, parsedHost.LocalHosts[0].DHCPOptionSetHash)
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

// Test convenience function which populates indexed subnets for an app.
func TestPopulateIndexedSubnetsForApp(t *testing.T) {
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)

	err := daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.2.0/24"
                },
                {
                    "subnet": "10.0.0.0/8"
                }
            ]
        }
	}`)
	require.NoError(t, err)

	app := &App{
		Daemons: []*Daemon{
			daemon,
		},
	}

	err = PopulateIndexedSubnets(app)
	require.NoError(t, err)

	indexedSubnets := app.Daemons[0].KeaDaemon.KeaDHCPDaemon.IndexedSubnets
	require.NotNil(t, indexedSubnets)
	require.Len(t, indexedSubnets.ByPrefix, 2)
	require.Contains(t, indexedSubnets.ByPrefix, "192.0.2.0/24")
	require.Contains(t, indexedSubnets.ByPrefix, "10.0.0.0/8")
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
			require.EqualValues(t, inputConfig.Map, outputConfig.Map)
		})
	}
}

// Test that KeaConfig and keaconfig.Map are parsed the same for empty string.
func TestKeaConfigIsAsKeaConfigMapForEmptyString(t *testing.T) {
	// Arrange
	json := ""

	// Act
	keaMap, errMap := keaconfig.NewFromJSON(json)
	keaConfig, errConfig := NewKeaConfigFromJSON(json)

	// Assert
	require.Error(t, errMap)
	require.Error(t, errConfig)

	require.Nil(t, keaMap)
	require.Nil(t, keaConfig)
}

// Test that KeaConfig and keaconfig.Map are parsed the same for empty JSON.
func TestKeaConfigIsAsKeaConfigMapForEmptyJSON(t *testing.T) {
	// Arrange
	json := "{}"

	// Act
	keaMap, errMap := keaconfig.NewFromJSON(json)
	keaConfig, errConfig := NewKeaConfigFromJSON(json)

	// Assert
	require.NoError(t, errMap)
	require.NoError(t, errConfig)

	require.EqualValues(t, keaMap, keaConfig.Map)
}

// Test that KeaConfig and keaconfig.Map are parsed the same for non-map JSON.
func TestKeaConfigIsAsKeaConfigMapNonMapJSON(t *testing.T) {
	// Arrange
	json := "foo"

	// Act
	keaMap, errMap := keaconfig.NewFromJSON(json)
	keaConfig, errConfig := NewKeaConfigFromJSON(json)

	// Assert
	require.Error(t, errMap)
	require.Error(t, errConfig)

	require.Nil(t, keaMap)
	require.Nil(t, keaConfig)
}

// Test that KeaConfig and keaconfig.Map are parsed the same for NULL from database.
func TestKeaConfigIsAsKeaConfigMapForNullFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Map
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

// Test that KeaConfig and keaconfig.Map are parsed the same for empty string from the database.
func TestKeaConfigIsAsKeaConfigMapForEmptyStringFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Map
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

// Test that KeaConfig and keaconfig.Map are parsed the same for empty JSON from the database.
func TestKeaConfigIsAsKeaConfigMapForEmptyJSONFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Map
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
	require.EqualValues(t, resMap.Config, resConfig.Config.Map)
}

// Test that KeaConfig and keaconfig.Map are parsed the same for JSON
// from a database containing single quote in value.
func TestKeaConfigIsAsKeaConfigMapForJSONWithSingleQuoteFromDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	var resMap struct {
		Config *keaconfig.Map
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
	require.EqualValues(t, resMap.Config, resConfig.Config.Map)
	rawConfig := *(*map[string]interface{})(resMap.Config)
	require.EqualValues(t, "b'r", rawConfig["foo"])
}

// Test storing a huge Kea configuration in the database.
func TestStoreHugeKeaConfigInDatabase(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// 50MB
	hugeValue := strings.Repeat("a", 50*1024*1024)
	rawConfig := map[string]interface{}{"Dhcp4": hugeValue}
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
	require.EqualValues(t, keaConfig.Map, addedDaemon.KeaDaemon.Config.Map)
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
