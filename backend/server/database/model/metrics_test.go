package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

func TestEmptyDatabaseMetrics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	metrics, err := GetCalculatedMetrics(db)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 0, metrics.AuthorizedMachines)
	require.EqualValues(t, 0, metrics.UnauthorizedMachines)
	require.EqualValues(t, 0, metrics.UnreachableMachines)
}

func TestFilledDatabaseMetrics(t *testing.T) {
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
