package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// This function adds two services, each including 5 Kea applications.
func addTestServices(t *testing.T, db *dbops.PgDB) []*Service {
	service1 := &Service{
		BaseService: BaseService{
			Name: "service1",
		},
	}
	service2 := &Service{
		BaseService: BaseService{
			Name: "service2",
		},
	}

	// Add 10 machines, each including a single Kea app.
	for i := 0; i < 10; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		a := &App{
			ID:          0,
			MachineID:   m.ID,
			Type:        KeaAppType,
			CtrlAddress: "cool.example.org",
			CtrlPort:    int64(1234 + i),
			Active:      true,
		}
		err = AddApp(db, a)
		require.NoError(t, err)

		// 5 apps added to service 1, and 5 added to service 2.
		if i%2 == 0 {
			service1.Apps = append(service1.Apps, a)
		} else {
			service2.Apps = append(service2.Apps, a)
		}
	}

	// Add the first service to the database. This one lacks the HA specific
	// information, simulating the non-HA service case.
	err := AddService(db, service1)
	require.NoError(t, err)

	// Service 2 holds HA specific information.
	service2.HAService = &BaseHAService{
		HAType:              "dhcp4",
		PrimaryID:           service2.Apps[0].ID,
		SecondaryID:         service2.Apps[1].ID,
		BackupID:            []int64{service2.Apps[2].ID, service2.Apps[3].ID},
		PrimaryStatusTime:   time.Now().UTC(),
		SecondaryStatusTime: time.Now().UTC(),
		PrimaryLastState:    "load-balancing",
		SecondaryLastState:  "syncing",
	}
	err = AddService(db, service2)
	require.NoError(t, err)

	// Return the services to the unit test.
	services := []*Service{service1, service2}
	return services
}

// Test that the base service can be updated.
func TestUpdateBaseService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Modify one of the services.
	service := services[0]
	service.Name = "funny name"
	err := UpdateBaseService(db, &service.BaseService)
	require.NoError(t, err)

	// Check that the new name is returned.
	returned, err := GetService(db, service.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, service.Name, returned.Name)
}

// Test that HA specific information can be updated for a service.
func TestUpdateBaseHAService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Modify HA information.
	service := services[1].HAService
	service.SecondaryLastState = "load-balancing"
	err := UpdateBaseHAService(db, service)
	require.NoError(t, err)

	// Check that the updated information is returned.
	returned, err := GetService(db, service.ServiceID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.NotNil(t, returned.HAService)
	require.Equal(t, service.SecondaryLastState, returned.HAService.SecondaryLastState)
}

// Test getting the service by id.
func TestGetServiceById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Get the first service. It should lack HA specific info.
	service, err := GetService(db, services[0].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Apps, 5)
	require.Nil(t, service.HAService)

	// Get the second service. It should include HA specific info.
	service, err = GetService(db, services[1].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Apps, 5)
	require.NotNil(t, service.HAService)
	require.Equal(t, "dhcp4", service.HAService.HAType)
	require.Equal(t, service.Apps[0].ID, service.HAService.PrimaryID)
	require.Equal(t, service.Apps[1].ID, service.HAService.SecondaryID)
	require.Len(t, service.HAService.BackupID, 2)
	require.Contains(t, service.HAService.BackupID, service.Apps[2].ID)
	require.Contains(t, service.HAService.BackupID, service.Apps[3].ID)
	require.False(t, service.HAService.PrimaryStatusTime.IsZero())
	require.False(t, service.HAService.SecondaryStatusTime.IsZero())
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "syncing", service.HAService.SecondaryLastState)
}

// Test getting services for an app.
func TestGetServicesByAppID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Get a service instance to which the forth application of the service1 belongs.
	appServices, err := GetServicesByAppID(db, services[0].Apps[3].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Validate that the service returned is the service1.
	service := appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[0].Name, service.Name)
	require.ElementsMatch(t, service.Apps, services[0].Apps)

	// Repeat the same test for the fifth application belonging to the service2.
	appServices, err = GetServicesByAppID(db, services[1].Apps[4].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Validate that the returned service is the service2.
	service = appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[1].Name, service.Name)
	require.ElementsMatch(t, service.Apps, services[1].Apps)

	// Finally, make one of the application shared between two services.
	err = AddAppToService(db, services[0].ID, services[1].Apps[0])
	require.NoError(t, err)

	// When querying the services for this app, both service1 and 2 should
	// be returned.
	appServices, err = GetServicesByAppID(db, services[1].Apps[0].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 2)

	require.Equal(t, services[0].Name, appServices[0].Name)
	require.Equal(t, services[1].Name, appServices[1].Name)
}

// Test getting the services including applications which operate on a
// given control address and port.
func TestGetServicesByAppCtrlAddressPort(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Get a service instance to which the forth application of the service1 belongs.
	appServices, err := GetServicesByAppCtrlAddressPort(db, services[0].Apps[3].CtrlAddress,
		services[0].Apps[3].CtrlPort)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Make sure that the first service was returned.
	service := appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[0].Name, service.Name)
	require.ElementsMatch(t, service.Apps, services[0].Apps)

	// Repeat the same test for the application belonging to the second service.
	appServices, err = GetServicesByAppCtrlAddressPort(db, services[1].Apps[3].CtrlAddress,
		services[1].Apps[3].CtrlPort)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Make sure that the second service was returned.
	service = appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[1].Name, service.Name)
	require.ElementsMatch(t, service.Apps, services[1].Apps)
}

// Test getting all services.
func TestGetAllServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// There should be two services returned.
	allServices, err := GetAllServices(db)
	require.NoError(t, err)
	require.Len(t, allServices, 2)

	// Services are sorted by ascending ID, so the first returned
	// service should be the one inserted.
	service := allServices[0]
	require.Len(t, service.Apps, 5)
	require.Nil(t, service.HAService)

	service = allServices[1]
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Apps, 5)

	// Make sure that the HA specific information was returned for the
	// second service.
	require.NotNil(t, service.HAService)
	require.Equal(t, "dhcp4", service.HAService.HAType)
}

// Test that the service can be deleted.
func TestDeleteService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Delete the second service.
	err := DeleteService(db, services[1].ID)
	require.NoError(t, err)

	// Try to get this service and make sure it is gone.
	service, err := GetService(db, services[1].ID)
	require.NoError(t, err)
	require.Nil(t, service)

	// Make sure it can be added back.
	service = services[1]
	service.ID = 0
	err = AddService(db, service)
	require.NoError(t, err)
}

// Test that a single app can be associated with the service.
func TestAddAppToService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Try to add an app which belongs to the second service to the
	// first service. It should succeed.
	err := AddAppToService(db, services[0].ID, services[1].Apps[0])
	require.NoError(t, err)

	// That service should not include 6 apps.
	service, err := GetService(db, services[0].ID)
	require.NoError(t, err)
	require.Len(t, service.Apps, 6)
}

// Test that a single app can be dissociated from the service.
func TestDeleteAppFromService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Delete association of one of the apps with the first service.
	ok, err := DeleteAppFromService(db, services[0].ID, services[0].Apps[0].ID)
	require.NoError(t, err)
	require.True(t, ok)

	// The service should now include 4 apps.
	service, err := GetService(db, 1)
	require.NoError(t, err)
	require.Len(t, service.Apps, 4)
}

// Test the convenience function checking if the service is new,
// i.e. hasn't yet been inserted into a database.
func TestIsServiceNew(t *testing.T) {
	// Create blank service lacking db ID. It should be considered new.
	s := Service{}
	require.True(t, s.IsNew())

	// Set ID and expect that the service is no longer new.
	s.ID = 100
	require.False(t, s.IsNew())
}
