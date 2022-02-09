package dumper

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Test that the factory creates proper, unique dump instances.
func TestFactoryProducesTheUniqueDumps(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)

	settings := agentcomm.AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := agentcomm.NewConnectedAgents(&settings, fec, []byte{}, []byte{}, []byte{})
	defer agents.Shutdown()

	factory := newFactory(db, m, agents)

	dumpTypeLookup := make(map[reflect.Type]bool)

	// Act
	dumps := factory.createAll()

	// Assert
	require.Len(t, dumps, 4)

	for _, dump := range dumps {
		dumpType := reflect.TypeOf(dump)
		_, ok := dumpTypeLookup[dumpType]
		require.False(t, ok, "duplicated type")
		dumpTypeLookup[dumpType] = true
	}
}

// Test that all created dumps are executed properly.
func TestAllProducedDumpsAreExecutedWithNoErrorForValidData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)

	settings := agentcomm.AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := agentcomm.NewConnectedAgents(&settings, fec, []byte{}, []byte{}, []byte{})
	defer agents.Shutdown()

	factory := newFactory(db, m, agents)
	dumps := factory.createAll()

	// Act
	for _, dump := range dumps {
		err := dump.Execute()

		// Assert
		require.NoError(t, err)
	}
}
