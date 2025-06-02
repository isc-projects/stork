package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Metrics should not crash even if the database is empty.
func TestEmptyDatabaseMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	metrics, err := GetCalculatedMetrics(db)

	// Assert
	require.NoError(t, err)
	require.Zero(t, metrics.AuthorizedMachines)
	require.Zero(t, metrics.UnauthorizedMachines)
	require.Zero(t, metrics.UnreachableMachines)
	require.Nil(t, metrics.SubnetMetrics)
	require.Nil(t, metrics.SharedNetworkMetrics)
}

// Metrics based on the machines should be properly calculated.
func TestFilledMachineDatabaseMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = AddMachine(db, &Machine{
		Address: "1", AgentPort: 1, Authorized: false,
	})
	_ = AddMachine(db, &Machine{
		Address: "2", AgentPort: 2, Authorized: false,
	})
	_ = AddMachine(db, &Machine{
		Address: "3", AgentPort: 3, Authorized: true,
	})
	_ = AddMachine(db, &Machine{
		Address: "4", AgentPort: 4, Authorized: false,
	})
	_ = AddMachine(db, &Machine{
		Address: "5", AgentPort: 5, Authorized: false, Error: "5",
	})
	_ = AddMachine(db, &Machine{
		Address: "6", AgentPort: 6, Authorized: true, Error: "6",
	})

	// Act
	metrics, err := GetCalculatedMetrics(db)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 2, metrics.AuthorizedMachines)
	require.EqualValues(t, 4, metrics.UnauthorizedMachines)
	require.EqualValues(t, 2, metrics.UnreachableMachines)
}

// Metrics per subnet should be properly calculated.
func TestFilledSubnetsDatabaseMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddSubnet(db, &Subnet{
		Prefix:          "3001:1::/64",
		AddrUtilization: Utilization(0.10),
		PdUtilization:   Utilization(0.15),
	})
	_ = AddSharedNetwork(db, &SharedNetwork{
		Name:   "alice",
		Family: 6,
		Subnets: []Subnet{
			{
				Prefix:          "3001:2::/64",
				AddrUtilization: Utilization(0.20),
				PdUtilization:   Utilization(0.25),
			},
		},
	})
	_ = AddSubnet(db, &Subnet{
		Prefix:          "192.168.2.1/32",
		AddrUtilization: Utilization(0.0),
		PdUtilization:   Utilization(0.0),
	})

	// Act
	metrics, err := GetCalculatedMetrics(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, metrics.SubnetMetrics, 3)

	require.EqualValues(t, "3001:1::/64", metrics.SubnetMetrics[0].Prefix)
	require.Empty(t, metrics.SubnetMetrics[0].SharedNetwork)
	require.EqualValues(t, 0.1, metrics.SubnetMetrics[0].AddrUtilization)
	require.EqualValues(t, 0.15, metrics.SubnetMetrics[0].PdUtilization)
	require.EqualValues(t, 6, metrics.SubnetMetrics[0].Family)

	require.EqualValues(t, "3001:2::/64", metrics.SubnetMetrics[1].Prefix)
	require.Equal(t, "alice", metrics.SubnetMetrics[1].SharedNetwork)
	require.EqualValues(t, 0.20, metrics.SubnetMetrics[1].AddrUtilization)
	require.EqualValues(t, 0.25, metrics.SubnetMetrics[1].PdUtilization)
	require.EqualValues(t, 6, metrics.SubnetMetrics[1].Family)

	require.EqualValues(t, "192.168.2.1/32", metrics.SubnetMetrics[2].Prefix)
	require.Empty(t, metrics.SubnetMetrics[2].SharedNetwork)
	require.Zero(t, metrics.SubnetMetrics[2].AddrUtilization)
	require.Zero(t, metrics.SubnetMetrics[2].PdUtilization)
	require.EqualValues(t, 4, metrics.SubnetMetrics[2].Family)
}

// Metrics per shared network should be properly calculated.
func TestFilledSharedNetworksDatabaseMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	// The "alice" share network is specified for both IPv4 and IPv6 families.
	// We do it to test if the metrics are calculated separately for each
	// family and the one family doesn't overwrite the other. This bug was
	// present in the previous implementation.
	_ = AddSharedNetwork(db, &SharedNetwork{
		Name:            "alice",
		AddrUtilization: 0.10,
		PdUtilization:   0.15,
		Family:          4,
	})
	_ = AddSharedNetwork(db, &SharedNetwork{
		Name:            "alice",
		AddrUtilization: 0.05,
		PdUtilization:   0.30,
		Family:          6,
	})
	_ = AddSharedNetwork(db, &SharedNetwork{
		Name:            "bob",
		AddrUtilization: 0.20,
		PdUtilization:   0.25,
		Family:          4,
	})
	_ = AddSharedNetwork(db, &SharedNetwork{
		Name:   "eva",
		Family: 6,
	})

	// Act
	metrics, err := GetCalculatedMetrics(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, metrics.SharedNetworkMetrics, 4)

	require.Empty(t, metrics.SharedNetworkMetrics[0].Prefix)
	require.EqualValues(t, "alice", metrics.SharedNetworkMetrics[0].SharedNetwork)
	require.EqualValues(t, 0.1, float64(metrics.SharedNetworkMetrics[0].AddrUtilization))
	require.EqualValues(t, 0.15, float64(metrics.SharedNetworkMetrics[0].PdUtilization))
	require.EqualValues(t, 4, metrics.SharedNetworkMetrics[0].Family)

	require.Empty(t, metrics.SharedNetworkMetrics[1].Prefix)
	require.EqualValues(t, "alice", metrics.SharedNetworkMetrics[1].SharedNetwork)
	require.EqualValues(t, 0.05, float64(metrics.SharedNetworkMetrics[1].AddrUtilization))
	require.EqualValues(t, 0.30, float64(metrics.SharedNetworkMetrics[1].PdUtilization))
	require.EqualValues(t, 6, metrics.SharedNetworkMetrics[1].Family)

	require.Empty(t, metrics.SharedNetworkMetrics[2].Prefix)
	require.EqualValues(t, "bob", metrics.SharedNetworkMetrics[2].SharedNetwork)
	require.EqualValues(t, 0.20, float64(metrics.SharedNetworkMetrics[2].AddrUtilization))
	require.EqualValues(t, 0.25, float64(metrics.SharedNetworkMetrics[2].PdUtilization))
	require.EqualValues(t, 4, metrics.SharedNetworkMetrics[2].Family)

	require.Empty(t, metrics.SharedNetworkMetrics[3].Prefix)
	require.EqualValues(t, "eva", metrics.SharedNetworkMetrics[3].SharedNetwork)
	require.Zero(t, metrics.SharedNetworkMetrics[3].AddrUtilization)
	require.Zero(t, metrics.SharedNetworkMetrics[3].PdUtilization)
	require.EqualValues(t, 6, metrics.SharedNetworkMetrics[3].Family)
}
