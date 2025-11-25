package dbmodel

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
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

	found := make([]bool, len(pts1))

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

// daemonsMatch compares two daemons and returns true if they match,
// false otherwise.
func daemonsMatch(daemon1, daemon2 *Daemon) bool {
	if daemon1.Pid != daemon2.Pid {
		return false
	}
	if daemon1.Name != daemon2.Name {
		return false
	}
	if daemon1.Active != daemon2.Active {
		return false
	}
	if daemon1.Version != daemon2.Version {
		return false
	}
	if daemon1.ExtendedVersion != daemon2.ExtendedVersion {
		return false
	}
	if daemon1.Uptime != daemon2.Uptime {
		return false
	}
	if daemon1.CreatedAt != daemon2.CreatedAt {
		return false
	}
	if daemon1.MachineID != daemon2.MachineID {
		return false
	}
	return accessPointArraysMatch(daemon1.AccessPoints, daemon2.AccessPoints)
}

// daemonArraysMatch compares two daemons arrays.  The two arrays may be
// ordered differently, as long as the elements in the array are identical,
// the two arrays are considered to match.  If so, this function returns
// true, false otherwise.
func daemonArraysMatch(daemonArray1, daemonArray2 []*Daemon) bool {
	if len(daemonArray1) != len(daemonArray2) {
		return false
	}

	if len(daemonArray1) == 0 {
		return true
	}

	found := make([]bool, len(daemonArray1))

	for i := 0; i < len(daemonArray1); i++ {
		for j := 0; j < len(daemonArray2); j++ {
			if daemonsMatch(daemonArray1[i], daemonArray2[j]) {
				found[i] = true
				break
			}
		}
	}

	for i := 0; i < len(daemonArray1); i++ {
		if !found[i] {
			return false
		}
	}
	return true
}

// Adds 20 test daemons.
func addTestDaemonsForServices(t *testing.T, db dbops.DBI) (daemons []*Daemon) {
	// Add 10 machines, each including two Kea daemons.
	for i := 0; i < 10; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		d1 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "cool.example.org",
				Port:    int64(1234 + i),
			},
		})
		err = AddDaemon(db, d1)
		require.NoError(t, err)

		d2 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "cool.example.org",
				Port:    int64(1234 + i + 100),
			},
		})
		err = AddDaemon(db, d2)
		require.NoError(t, err)

		daemons = append(daemons, d1, d2)
	}
	return daemons
}

// This function adds four services and ten daemons. It associates each daemon
// with two services.
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

	service3 := &Service{
		BaseService: BaseService{
			Name: "service3",
		},
	}

	service4 := &Service{
		BaseService: BaseService{
			Name: "service4",
		},
	}

	daemons := addTestDaemonsForServices(t, db)
	for i := range daemons {
		// 5 daemons added to service 1 and 3. 5 added to service 2 and 4.
		switch i % 4 {
		case 0:
			service1.Daemons = append(service1.Daemons, daemons[i])
			service3.Daemons = append(service3.Daemons, daemons[i])
		case 2:
			service2.Daemons = append(service2.Daemons, daemons[i])
			service4.Daemons = append(service4.Daemons, daemons[i])
		}
	}

	// Add the first service to the database. This one lacks the HA specific
	// information, simulating the non-HA service case.
	err := AddService(db, service1)
	require.NoError(t, err)

	// Service 2 holds HA specific information.
	commInterrupted := make([]bool, 2)
	commInterrupted[0] = true
	commInterrupted[1] = false
	service2.HAService = &BaseHAService{
		HAType:                      "dhcp4",
		Relationship:                "server1",
		PrimaryID:                   service2.Daemons[0].ID,
		SecondaryID:                 service2.Daemons[1].ID,
		BackupID:                    []int64{service2.Daemons[2].ID, service2.Daemons[3].ID},
		PrimaryStatusCollectedAt:    time.Now(),
		SecondaryStatusCollectedAt:  time.Now(),
		PrimaryReachable:            true,
		SecondaryReachable:          true,
		PrimaryLastState:            "load-balancing",
		SecondaryLastState:          "syncing",
		PrimaryLastScopes:           []string{"server1", "server2"},
		SecondaryLastScopes:         []string{},
		PrimaryLastFailoverAt:       time.Now(),
		PrimaryCommInterrupted:      &commInterrupted[0],
		SecondaryCommInterrupted:    &commInterrupted[1],
		PrimaryConnectingClients:    1,
		SecondaryConnectingClients:  0,
		PrimaryUnackedClients:       2,
		SecondaryUnackedClients:     0,
		PrimaryUnackedClientsLeft:   6,
		SecondaryUnackedClientsLeft: 0,
		PrimaryAnalyzedPackets:      9,
		SecondaryAnalyzedPackets:    0,
	}
	err = AddService(db, service2)
	require.NoError(t, err)

	err = AddService(db, service3)
	require.NoError(t, err)

	err = AddService(db, service4)
	require.NoError(t, err)

	// Return the services to the unit test.
	services := []*Service{service1, service2, service3, service4}
	return services
}

// Test that the base service can be updated.
func TestUpdateBaseService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// Modify one of the services.
	service := services[0]
	service.Name = "funny name"

	// Remember the creation time so it can be compared after the update.
	createdAt := service.CreatedAt
	require.NotZero(t, createdAt)

	// Reset creation time to ensure it is not modified during the update.
	service.CreatedAt = time.Time{}
	err := UpdateBaseService(db, &service.BaseService)
	require.NoError(t, err)

	// Check that the new name is returned.
	returned, err := GetDetailedService(db, service.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Equal(t, service.Name, returned.Name)
	require.Equal(t, createdAt, returned.CreatedAt)
}

// Test that HA specific information can be updated for a service.
func TestUpdateBaseHAService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

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
	require.GreaterOrEqual(t, len(services), 4)

	// Update the existing service by adding HA specific information to it.
	services[0].HAService = &BaseHAService{
		HAType:                     "dhcp4",
		Relationship:               "server1",
		PrimaryID:                  services[0].Daemons[0].ID,
		SecondaryID:                services[0].Daemons[1].ID,
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
	require.NotNil(t, service.HAService)
	require.Equal(t, daemonname.DHCPv4, service.HAService.HAType)
	require.Equal(t, "server1", service.HAService.Relationship)
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
	require.GreaterOrEqual(t, len(services), 4)

	// Get the first service. It should lack HA specific info.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Daemons, 5)
	require.Nil(t, service.HAService)

	// Get the second service. It should include HA specific info.
	service, err = GetDetailedService(db, services[1].ID)
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Daemons, 5)
	require.NotNil(t, service.HAService)
	require.Equal(t, daemonname.DHCPv4, service.HAService.HAType)
	require.Equal(t, service.Daemons[0].ID, service.HAService.PrimaryID)
	require.Equal(t, service.Daemons[1].ID, service.HAService.SecondaryID)
	require.Len(t, service.HAService.BackupID, 2)
	require.Contains(t, service.HAService.BackupID, service.Daemons[2].ID)
	require.Contains(t, service.HAService.BackupID, service.Daemons[3].ID)
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "syncing", service.HAService.SecondaryLastState)
	require.False(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())
	require.NotNil(t, *service.HAService.PrimaryCommInterrupted)
	require.True(t, *service.HAService.PrimaryCommInterrupted)
	require.NotNil(t, *service.HAService.SecondaryCommInterrupted)
	require.False(t, *service.HAService.SecondaryCommInterrupted)
	require.EqualValues(t, 1, service.HAService.PrimaryConnectingClients)
	require.Zero(t, service.HAService.SecondaryConnectingClients)
	require.EqualValues(t, 2, service.HAService.PrimaryUnackedClients)
	require.Zero(t, service.HAService.SecondaryUnackedClients)
	require.EqualValues(t, 6, service.HAService.PrimaryUnackedClientsLeft)
	require.Zero(t, service.HAService.SecondaryUnackedClientsLeft)
	require.EqualValues(t, 9, service.HAService.PrimaryAnalyzedPackets)
	require.Zero(t, service.HAService.SecondaryAnalyzedPackets)
}

// Test getting services for a daemon.
func TestGetServicesByDaemonID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// Get a service instance to which the forth daemons of the service1 belongs.
	daemonServices, err := GetDetailedServicesByDaemonID(db, services[0].Daemons[3].ID)
	require.NoError(t, err)
	require.Len(t, daemonServices, 2)
	sort.Slice(daemonServices, func(i, j int) bool {
		return daemonServices[i].Name < daemonServices[j].Name
	})
	require.Len(t, daemonServices[0].Daemons[0].AccessPoints, 1)

	// Validate that the service returned is the service1.
	service := daemonServices[0]
	require.Len(t, service.Daemons, 5)
	require.Equal(t, services[0].Name, service.Name)
	require.True(t, daemonArraysMatch(service.Daemons, services[0].Daemons))

	// Repeat the same test for the fifth daemon belonging to the service2.
	daemonServices, err = GetDetailedServicesByDaemonID(db, services[1].Daemons[4].ID)
	sort.Slice(daemonServices, func(i, j int) bool {
		return daemonServices[i].Name < daemonServices[j].Name
	})
	require.NoError(t, err)
	require.Len(t, daemonServices, 2)

	// Validate that the returned service is the service2.
	service = daemonServices[0]
	require.Len(t, service.Daemons, 5)
	require.Equal(t, services[1].Name, service.Name)
	require.True(t, daemonArraysMatch(service.Daemons, services[1].Daemons))

	// Second one is service4.
	service = daemonServices[1]
	require.Len(t, service.Daemons, 5)
	require.Equal(t, services[3].Name, service.Name)
	require.True(t, daemonArraysMatch(service.Daemons, services[3].Daemons))

	// Finally, make one of the daemons shared between two services.
	err = AddDaemonToService(db, services[0].ID, services[1].Daemons[0])
	require.NoError(t, err)

	// When querying the services for this daemon service1, 2 and 4 should
	// be returned.
	daemonServices, err = GetDetailedServicesByDaemonID(db, services[1].Daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, daemonServices, 3)
	sort.Slice(daemonServices, func(i, j int) bool {
		return daemonServices[i].Name < daemonServices[j].Name
	})

	require.Equal(t, services[0].Name, daemonServices[0].Name)
	require.Equal(t, services[1].Name, daemonServices[1].Name)
	require.Equal(t, services[3].Name, daemonServices[2].Name)
}

// Test that it is possible to get daemons by type and get the services
// returned along with them.
func TestGetDaemonWithServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	daemons, err := GetDaemonsByName(db, daemonname.DHCPv4)
	require.NoError(t, err)
	require.Len(t, daemons, 10)

	// Make sure that all returned daemons contain references to the services.
	for _, daemon := range daemons {
		require.Len(t, daemon.Services, 2)
	}
}

// Test getting all services.
func TestGetAllServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// There should be four services returned.
	allServices, err := GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, allServices, 4)
	sort.Slice(allServices, func(i, j int) bool {
		return allServices[i].Name < allServices[j].Name
	})

	service := allServices[0]
	require.Len(t, service.Daemons, 5)
	require.Nil(t, service.HAService)

	service = allServices[1]
	require.NoError(t, err)
	require.NotNil(t, service)
	require.Len(t, service.Daemons, 5)

	// Make sure that the HA specific information was returned for the
	// second service.
	require.NotNil(t, service.HAService)
	require.Equal(t, daemonname.DHCPv4, service.HAService.HAType)

	service = allServices[2]
	require.Len(t, service.Daemons, 5)
	require.Nil(t, service.HAService)

	service = allServices[3]
	require.Len(t, service.Daemons, 5)
	require.Nil(t, service.HAService)
}

// Test that the service can be deleted.
func TestDeleteService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

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

// Test that a single daemon can be associated with the service.
func TestAddDaemonToService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// Try to add a daemon which belongs to the second service to the
	// first service. It should succeed.
	err := AddDaemonToService(db, services[0].ID, services[1].Daemons[0])
	require.NoError(t, err)

	// That service should now include 6 apps.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.Len(t, service.Daemons, 6)
}

// Test that a single daemon can be dissociated from the service.
func TestDeleteDaemonFromService(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// Delete association of one of the daemons with the first service.
	ok, err := DeleteDaemonFromService(db, services[0].ID, services[0].Daemons[0].ID)
	require.NoError(t, err)
	require.True(t, ok)

	// The service should now include 4 daemons. One has been removed.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.Len(t, service.Daemons, 4)
}

// Test that a daemon can be dissociated from all services.
func TestDeleteDaemonFromServices(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	require.GreaterOrEqual(t, len(services), 4)

	// Delete association of one of the daemons with the first service.
	rows, err := DeleteDaemonFromServices(db, services[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, rows)

	// First and third service should now include only 4 daemons.
	service, err := GetDetailedService(db, services[0].ID)
	require.NoError(t, err)
	require.Len(t, service.Daemons, 4)

	service, err = GetDetailedService(db, services[2].ID)
	require.NoError(t, err)
	require.Len(t, service.Daemons, 4)
}

// Test that multiple services can be added/updated and associated with a
// daemon within a single transaction.
func TestCommitServicesIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	daemons := addTestDaemonsForServices(t, db)

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

	// Add first two services into db and associate with the first daemon.
	err := CommitServicesIntoDB(db, services[:2], daemons[0])
	require.NoError(t, err)

	// Get the services. There should be two in the database.
	returned, err := GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	for i := range returned {
		// Make sure they are both associated with our daemon.
		require.Len(t, returned[i].Daemons, 1)
		require.Equal(t, services[i].Name, returned[i].Name)
		require.EqualValues(t, daemons[0].ID, returned[i].Daemons[0].ID)
	}

	// This time commit daemon #2 and #3 into db and associate them with the
	// second daemon.
	err = CommitServicesIntoDB(db, services[1:3], daemons[1])
	require.NoError(t, err)

	// Get the services snapshot from the db again.
	returned, err = GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, returned, 3)

	// The first and third daemon should be associated with one daemon and the
	// second one should be associated with both daemons.
	require.Len(t, returned[0].Daemons, 1)
	require.Len(t, returned[1].Daemons, 2)
	require.Len(t, returned[2].Daemons, 1)
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

// Verifies that the correct HA state is returned by the service for
// the particular daemon ID.
func TestGetDaemonHAState(t *testing.T) {
	service := Service{
		HAService: &BaseHAService{
			HAType:             "dhcp4",
			PrimaryID:          1,
			SecondaryID:        2,
			BackupID:           []int64{3, 4},
			PrimaryLastState:   "load-balancing",
			SecondaryLastState: "syncing",
		},
	}

	require.Equal(t, "load-balancing", service.GetDaemonHAState(1))
	require.Equal(t, "syncing", service.GetDaemonHAState(2))
	require.Equal(t, "backup", service.GetDaemonHAState(3))
	require.Equal(t, "backup", service.GetDaemonHAState(4))
	require.Empty(t, service.GetDaemonHAState(5))

	service.HAService = nil
	require.Empty(t, service.GetDaemonHAState(1))
}

// Test that the partner's failure time is returned correctly.
func TestGetPartnerHAFailureTime(t *testing.T) {
	// If this is not HA service, the time returned should be zero.
	service := Service{}
	failureTime := service.GetPartnerHAFailureTime(1)
	require.Zero(t, failureTime)

	primaryFailoverAt := time.Date(2020, 6, 4, 11, 32, 0, 0, time.UTC)
	service.HAService = &BaseHAService{
		HAType:                "dhcp4",
		PrimaryID:             1,
		SecondaryID:           2,
		BackupID:              []int64{3, 4},
		PrimaryLastState:      "load-balancing",
		SecondaryLastState:    "load-balancing",
		PrimaryLastFailoverAt: primaryFailoverAt,
	}

	// Specify primary id, which should return its failure time based
	// on the failover time of the secondary. This should be zero.
	failureTime = service.GetPartnerHAFailureTime(1)
	require.Zero(t, failureTime)
	// When specifying secondary id, the failure time returned should be
	// the primary's failover time.
	failureTime = service.GetPartnerHAFailureTime(2)
	require.Equal(t, primaryFailoverAt, failureTime)
}

// Tests that passive HA daemons are selected properly when HA works correctly.
func TestGetPassiveHADaemonIDs(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	haService := services[1]
	haService.HAService.PrimaryLastState = HAStateReady
	haService.HAService.SecondaryLastState = HAStateReady
	_ = UpdateService(db, haService)

	// Act
	daemons, err := GetPassiveHADaemonIDs(db)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, haService.HAService)
	// 1 secondary, 2 backups
	require.Len(t, daemons, 3)
	require.Contains(t, daemons, haService.HAService.SecondaryID)
	require.Contains(t, daemons, haService.HAService.BackupID[0])
	require.Contains(t, daemons, haService.HAService.BackupID[1])
}

// Tests that passive HA daemons are selected properly when HA daemons are
// unreachable.
func TestGetPassiveHAUnreachableDaemonIDs(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	haService := services[1]
	haService.HAService.PrimaryReachable = false
	haService.HAService.SecondaryReachable = false
	_ = UpdateService(db, haService)

	// Act
	daemons, err := GetPassiveHADaemonIDs(db)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, haService.HAService)
	// Both servers are non-operational. Fallback to primary.
	// 1 secondary, 2 backups
	require.Len(t, daemons, 3)
}

// Tests that passive HA daemons are selected properly when a primary
// daemon isn't operational.
func TestGetPassiveHADaemonIDsPrimaryIsNotOperational(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	services := addTestServices(t, db)
	haService := services[1]
	haService.HAService.PrimaryLastState = HAStateSyncing
	haService.HAService.SecondaryLastState = HAStateReady
	_ = UpdateService(db, haService)

	// Act
	daemons, err := GetPassiveHADaemonIDs(db)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, haService.HAService)
	// 1 primary, 2 backups
	require.Len(t, daemons, 3)
	require.Contains(t, daemons, haService.HAService.PrimaryID)
	require.Contains(t, daemons, haService.HAService.BackupID[0])
	require.Contains(t, daemons, haService.HAService.BackupID[1])
}
