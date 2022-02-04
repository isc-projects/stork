package dbmodel

import (
	"strings"
	"testing"

	"github.com/go-pg/pg/v10"
	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dbtest "isc.org/stork/server/database/test"
)

// Test that KeaConfig is constructed properly.
func TestNewKeaConfig(t *testing.T) {
	// Act
	configNil := NewKeaConfig(nil)
	configEmpty := NewKeaConfig(&map[string]interface{}{})
	configFilled := NewKeaConfig(&map[string]interface{}{"Dhcp4": "foo"})
	root, ok := configFilled.GetRootName()

	// Assert
	require.Nil(t, configNil)
	require.EqualValues(t, map[string]interface{}{}, *configEmpty.Map)
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
			},
			{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			},
		},
	}

	parsedNetwork, err := NewSharedNetworkFromKea(&rawNetwork, 6)
	require.NoError(t, err)
	require.NotNil(t, parsedNetwork)
	require.Equal(t, "foo", parsedNetwork.Name)
	require.EqualValues(t, 6, parsedNetwork.Family)
	require.Len(t, parsedNetwork.Subnets, 2)

	require.Zero(t, parsedNetwork.Subnets[0].ID)
	require.Equal(t, "2001:db8:2::/64", parsedNetwork.Subnets[0].Prefix)
	require.Zero(t, parsedNetwork.Subnets[1].ID)
	require.Equal(t, "2001:db8:1::/64", parsedNetwork.Subnets[1].Prefix)
}

// Test that subnets within a shared network are verified to catch
// those which family is not matching with the shared network family.
func TestNewSharedNetworkFromKeaFamilyClash(t *testing.T) {
	rawNetwork := map[string]interface{}{
		"name": "foo",
		"subnet4": []map[string]interface{}{
			{
				"id":     1,
				"subnet": "192.0.2.0/24",
			},
		},
		"subnet6": []map[string]interface{}{
			{
				"id":     2,
				"subnet": "2001:db8:1::/64",
			},
		},
	}

	parsedNetwork, err := NewSharedNetworkFromKea(&rawNetwork, 4)
	require.Error(t, err)
	require.Nil(t, parsedNetwork)
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
				"prefix":        "2001:db8:1:1::",
				"prefix-len":    96,
				"delegated-len": 120,
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

	parsedSubnet, err := NewSubnetFromKea(&rawSubnet)
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

	require.Len(t, parsedSubnet.Hosts, 1)
	require.Len(t, parsedSubnet.Hosts[0].HostIdentifiers, 2)
	require.Equal(t, "duid", parsedSubnet.Hosts[0].HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, parsedSubnet.Hosts[0].HostIdentifiers[0].Value)
	require.Equal(t, "hw-address", parsedSubnet.Hosts[0].HostIdentifiers[1].Type)
	require.Equal(t, []byte{1, 1, 1, 1, 1, 1}, parsedSubnet.Hosts[0].HostIdentifiers[1].Value)

	require.Len(t, parsedSubnet.Hosts[0].IPReservations, 4)
	require.Equal(t, "2001:db8:1::1", parsedSubnet.Hosts[0].IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedSubnet.Hosts[0].IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedSubnet.Hosts[0].IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedSubnet.Hosts[0].IPReservations[3].Address)
}

// Verifies that the host instance can be created by parsing Kea
// configuration.
func TestNewHostFromKea(t *testing.T) {
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
		"hostname": "hostname.example.org",
	}

	parsedHost, err := NewHostFromKea(&rawHost)
	require.NoError(t, err)
	require.NotNil(t, parsedHost)

	require.Len(t, parsedHost.HostIdentifiers, 1)
	require.Equal(t, "duid", parsedHost.HostIdentifiers[0].Type)
	require.Len(t, parsedHost.IPReservations, 4)
	require.Equal(t, "2001:db8:1::1", parsedHost.IPReservations[0].Address)
	require.Equal(t, "2001:db8:1::2", parsedHost.IPReservations[1].Address)
	require.Equal(t, "3000:1::/64", parsedHost.IPReservations[2].Address)
	require.Equal(t, "3000:2::/64", parsedHost.IPReservations[3].Address)
	require.Equal(t, "hostname.example.org", parsedHost.Hostname)
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

// Test that KeaConfig and keaconfig.Map are parsed the same for NULL from database.
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

// Test that KeaConfig and keaconfig.Map are parsed the same for empty JSON from database.
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

// Test that KeaConfig and keaconfig.Map are storing nil in the database as NULL.
func TestKeaConfigIsAsKeaConfigMapForStoringNilInDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, err := db.Exec("CREATE TABLE jsons (id serial PRIMARY KEY, config jsonb)")
	require.NoError(t, err)

	var resMap struct {
		Config *keaconfig.Map
	}

	var resConfig struct {
		Config *KeaConfig
	}

	var resCount struct {
		Count int64
	}

	query := `SELECT COUNT(*) FROM jsons GROUP BY config LIMIT 1`

	// Act
	_, errInsertMap := db.Model(&resMap).Table("jsons").Insert()
	_, errInsertConfig := db.Model(&resConfig).Table("jsons").Insert()
	_, errCount := db.QueryOne(&resCount, query)

	// Assert
	require.NoError(t, errInsertMap)
	require.NoError(t, errInsertConfig)
	require.NoError(t, errCount)

	require.Nil(t, resMap.Config)
	require.Nil(t, resConfig.Config)
	require.EqualValues(t, 2, resCount.Count)
}

// Test store a huge Kea configuration in the database.
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

	_, err = AddApp(db, &App{
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons:   []*Daemon{daemon},
	})
	require.NoError(t, err)
}
