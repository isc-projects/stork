package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// accessPointsMatch compares two access points and returns true if they
// match, false otherwise.
func accessPointsMatch(pt1, pt2 *AccessPoint) bool {
	if pt1.Type != pt2.Type {
		return false
	}
	if pt1.Address != pt2.Address {
		return false
	}
	if pt1.Port != pt2.Port {
		return false
	}
	if pt1.Key != pt2.Key {
		return false
	}
	return true
}

// accessPointArraysMatch compares two access point arrays and returns true
// if they match, false otherwise.
func accessPointArraysMatch(pts1, pts2 []*AccessPoint) bool {
	if len(pts1) != len(pts2) {
		return false
	}

	if len(pts1) == 0 {
		return true
	}

	var found = make([]bool, len(pts1))

	for i := 0; i < len(pts1); i++ {
		for j := 0; j < len(pts2); j++ {
			if accessPointsMatch(pts1[i], pts2[j]) {
				found[i] = true
				break
			}
		}
	}

	for i := 0; i < len(pts1); i++ {
		if !found[i] {
			return false
		}
	}

	return true
}

// appsMatch compares two application and returns true if they match,
// false otherwise.
func appsMatch(app1, app2 *App) bool {
	if app1.ID != app2.ID {
		return false
	}
	if app1.CreatedAt != app2.CreatedAt {
		return false
	}
	if app1.MachineID != app2.MachineID {
		return false
	}
	if app1.Type != app2.Type {
		return false
	}
	if app1.Active != app2.Active {
		return false
	}
	return accessPointArraysMatch(app1.AccessPoints, app2.AccessPoints)
}

// appArraysMatch compares two application arrays.  The two arrays may be
// ordered differently, as long as the elements in the array are identical,
// the two arrays are considered to match.  If so, this function returns
// true, false otherwise.
func appArraysMatch(appArray1, appArray2 []*App) bool {
	if len(appArray1) != len(appArray2) {
		return false
	}

	if len(appArray1) == 0 {
		return true
	}

	var found = make([]bool, len(appArray1))

	for i := 0; i < len(appArray1); i++ {
		for j := 0; j < len(appArray2); j++ {
			if appsMatch(appArray1[i], appArray2[j]) {
				found[i] = true
				break
			}
		}
	}

	for i := 0; i < len(appArray1); i++ {
		if !found[i] {
			return false
		}
	}
	return true
}

// Adds 10 test apps.
func addTestApps(t *testing.T, db *dbops.PgDB) (apps []*App) {
	// Add 10 machines, each including a single Kea app.
	for i := 0; i < 10; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		var accessPoints []*AccessPoint
		accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", int64(1234+i))
		a := &App{
			ID:           0,
			MachineID:    m.ID,
			Type:         AppTypeKea,
			Active:       true,
			AccessPoints: accessPoints,
		}

		err = AddApp(db, a)
		require.NoError(t, err)

		apps = append(apps, a)
	}
	return apps
}

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

	apps := addTestApps(t, db)
	for i := range apps {
		// 5 apps added to service 1, and 5 added to service 2.
		if i%2 == 0 {
			service1.Apps = append(service1.Apps, apps[i])
		} else {
			service2.Apps = append(service2.Apps, apps[i])
		}
	}

	// Add the first service to the database. This one lacks the HA specific
	// information, simulating the non-HA service case.
	err := AddService(db, service1)
	require.NoError(t, err)

	// Service 2 holds HA specific information.
	service2.HAService = &BaseHAService{
		HAType:                     "dhcp4",
		PrimaryID:                  service2.Apps[0].ID,
		SecondaryID:                service2.Apps[1].ID,
		BackupID:                   []int64{service2.Apps[2].ID, service2.Apps[3].ID},
		PrimaryStatusCollectedAt:   time.Now(),
		SecondaryStatusCollectedAt: time.Now(),
		PrimaryLastState:           "load-balancing",
		SecondaryLastState:         "syncing",
		PrimaryLastScopes:          []string{"server1", "server2"},
		SecondaryLastScopes:        []string{},
		PrimaryLastFailoverAt:      time.Now(),
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
	returned, err := GetDetailedService(db, service.ID)
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
	returned, err := GetDetailedService(db, service.ServiceID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.NotNil(t, returned.HAService)
	require.Equal(t, service.SecondaryLastState, returned.HAService.SecondaryLastState)
}

// Test that the entire service information can be updated.
func TestUpdateService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Update the existing service by adding HA specific information to it.
	services[0].HAService = &BaseHAService{
		HAType:                     "dhcp4",
		PrimaryID:                  services[0].Apps[0].ID,
		SecondaryID:                services[0].Apps[1].ID,
		PrimaryStatusCollectedAt:   storkutil.UTCNow(),
		SecondaryStatusCollectedAt: storkutil.UTCNow(),
		PrimaryLastState:           "load-balancing",
		SecondaryLastState:         "syncing",
		PrimaryLastScopes:          []string{"server1"},
		SecondaryLastScopes:        []string{"server2"},
	}
	err := UpdateService(db, services[0])
	require.NoError(t, err)

	// Make sure that the HA specific information was attached.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Equal(t, "ha_dhcp", service.ServiceType)
	require.NotNil(t, service.HAService)
	require.Equal(t, "dhcp4", service.HAService.HAType)
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "syncing", service.HAService.SecondaryLastState)
	require.Len(t, service.HAService.PrimaryLastScopes, 1)
	require.Equal(t, "server1", service.HAService.PrimaryLastScopes[0])
	require.Len(t, service.HAService.SecondaryLastScopes, 1)
	require.Equal(t, "server2", service.HAService.SecondaryLastScopes[0])

	// Try to update HA specific information.
	service.HAService.SecondaryLastState = "load-balancing"
	err = UpdateService(db, service)
	require.NoError(t, err)

	service, err = GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.NotNil(t, service.HAService)
	require.Equal(t, "load-balancing", service.HAService.SecondaryLastState)
}

// Test getting the service by id.
func TestGetServiceById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Get the first service. It should lack HA specific info.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Apps, 5)
	require.Nil(t, service.HAService)

	// Get the second service. It should include HA specific info.
	service, err = GetDetailedService(db, services[1].ID)
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
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "syncing", service.HAService.SecondaryLastState)
	require.False(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())
}

// Test getting services for an app.
func TestGetServicesByAppID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// Get a service instance to which the forth application of the service1 belongs.
	appServices, err := GetDetailedServicesByAppID(db, services[0].Apps[3].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Validate that the service returned is the service1.
	service := appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[0].Name, service.Name)
	require.True(t, appArraysMatch(service.Apps, services[0].Apps))

	// Repeat the same test for the fifth application belonging to the service2.
	appServices, err = GetDetailedServicesByAppID(db, services[1].Apps[4].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 1)

	// Validate that the returned service is the service2.
	service = appServices[0]
	require.Len(t, service.Apps, 5)
	require.Equal(t, services[1].Name, service.Name)
	require.True(t, appArraysMatch(service.Apps, services[1].Apps))

	// Finally, make one of the application shared between two services.
	err = AddAppToService(db, services[0].ID, services[1].Apps[0])
	require.NoError(t, err)

	// When querying the services for this app, both service1 and 2 should
	// be returned.
	appServices, err = GetDetailedServicesByAppID(db, services[1].Apps[0].ID)
	require.NoError(t, err)
	require.Len(t, appServices, 2)

	require.Equal(t, services[0].Name, appServices[0].Name)
	require.Equal(t, services[1].Name, appServices[1].Name)
}

// Test getting all services.
func TestGetAllServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 2)

	// There should be two services returned.
	allServices, err := GetDetailedAllServices(db)
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
	service, err := GetDetailedService(db, services[1].ID)
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

	// That service should now include 6 apps.
	service, err := GetDetailedService(db, services[0].ID)
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
	service, err := GetDetailedService(db, 1)
	require.NoError(t, err)
	require.Len(t, service.Apps, 4)
}

// Test that multiple services can be added/updated and associated with an
// app within a single transaction.
func TestCommitServicesIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestApps(t, db)

	services := []Service{
		{
			BaseService: BaseService{
				Name: "service1",
			},
		},
		{
			BaseService: BaseService{
				Name: "service2",
			},
		},
		{
			BaseService: BaseService{
				Name: "service3",
			},
		},
	}

	// Add first two services into db and associate with the first app.
	err := CommitServicesIntoDB(db, services[:2], apps[0])
	require.NoError(t, err)

	// Get the services. There should be two in the database.
	returned, err := GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	for i := range returned {
		// Make sure they are both associated with our app.
		require.Len(t, returned[i].Apps, 1)
		require.Equal(t, services[i].Name, returned[i].Name)
		require.EqualValues(t, apps[0].ID, returned[i].Apps[0].ID)
	}

	// This time commit app #2 and #3 into db and associate them with the
	// second app.
	err = CommitServicesIntoDB(db, services[1:3], apps[1])
	require.NoError(t, err)

	// Get the services shanpshot from the db again.
	returned, err = GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, returned, 3)

	// The first and third app should be associated with one app and the
	// second one should be associated with both apps.
	require.Len(t, returned[0].Apps, 1)
	require.Len(t, returned[1].Apps, 2)
	require.Len(t, returned[2].Apps, 1)
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
