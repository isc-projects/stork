package dump_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dumppkg "isc.org/stork/server/dumper/dump"
)

// Test creating a dump of the HA services.
func TestServicesDumpExecute(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app.
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "cool.example.org", "", int64(1234), false)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	// Add first service.
	service := &dbmodel.Service{
		BaseService: dbmodel.BaseService{
			ID:      123,
			Daemons: app.Daemons,
			Name:    "foo",
		},
		HAService: &dbmodel.BaseHAService{
			HAType:           "dhcp4",
			PrimaryLastState: "load-balancing",
		},
	}
	err = dbmodel.AddService(db, service)
	require.NoError(t, err)

	// Add second service.
	service = &dbmodel.Service{
		BaseService: dbmodel.BaseService{
			ID:      234,
			Daemons: app.Daemons,
			Name:    "bar",
		},
		HAService: &dbmodel.BaseHAService{
			HAType:           "dhcp4",
			PrimaryLastState: "hot-standby",
		},
	}
	err = dbmodel.AddService(db, service)
	require.NoError(t, err)

	// Create the services dump for the machine.
	dump := dumppkg.NewServicesDump(db, machine)
	err = dump.Execute()
	require.NoError(t, err)

	// Each service dump should be in its own artifact.
	require.EqualValues(t, 2, dump.GetArtifactsNumber())
	artifacts := make([]dumppkg.StructArtifact, 2)

	// Get the artifacts and sort them by name.
	for i := 0; i < 2; i++ {
		artifacts[i] = dump.GetArtifact(i).(dumppkg.StructArtifact)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].GetName() < artifacts[j].GetName()
	})

	// Validate first artifact.
	require.Equal(t, "s-123-foo", artifacts[0].GetName())
	artifactContent := artifacts[0].GetStruct()
	require.NotNil(t, artifactContent)
	service, ok := artifactContent.(*dbmodel.Service)
	require.True(t, ok)
	require.NotNil(t, service)
	require.Equal(t, "foo", service.Name)
	require.NotNil(t, service.HAService)
	require.Equal(t, "dhcp4", service.HAService.HAType)
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)

	// Validate second artifact.
	require.Equal(t, "s-234-bar", artifacts[1].GetName())
	artifactContent = artifacts[1].GetStruct()
	require.NotNil(t, artifactContent)
	service, ok = artifactContent.(*dbmodel.Service)
	require.True(t, ok)
	require.NotNil(t, service)
	require.Equal(t, "bar", service.Name)
	require.NotNil(t, service.HAService)
	require.Equal(t, "dhcp4", service.HAService.HAType)
	require.Equal(t, "hot-standby", service.HAService.PrimaryLastState)
}

// Test dumping the services when there are no services available.
func TestServicesDumpExecuteNoServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Create the services dump for the machine.
	dump := dumppkg.NewServicesDump(db, machine)
	err = dump.Execute()
	require.NoError(t, err)

	// There are no services so, there should be no artifacts.
	require.EqualValues(t, 0, dump.GetArtifactsNumber())
}
