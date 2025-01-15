package dnsop

import (
	"testing"

	"github.com/stretchr/testify/require"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	appstest "isc.org/stork/server/apps/test"
	dbtest "isc.org/stork/server/database/test"
)

func TestNewManager(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: nil,
	})
	require.NotNil(t, manager)
}

func TestFetchZones(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := agentcommtest.NewKeaFakeAgents()
	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	})
	require.NotNil(t, manager)

	err := manager.FetchZones()
	require.Nil(t, err)
}
