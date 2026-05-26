package restservice

import (
	"context"
	"math"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Verify that [convertLeaseToRestAPI] returns an error when called with
// a nil [dbmodel.Lease].
func TestConvertLeaseFromRestAPIWithNilLease(t *testing.T) {
	result, err := convertLeaseToRestAPI(nil)

	require.Nil(t, result)
	require.ErrorContains(t, err, "nil")
}

// Verify that [convertLeaseToRestAPI] returns en error when called with
// a [dbmodel.Lease] which has a CLTT larger than will fit in a (signed) int64.
func TestConvertLeaseFromRestAPIWithCLTTTooBig(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 + 1,
		},
	}
	result, err := convertLeaseToRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "CLTT")
}

// Verify that [convertLeaseToRestAPI] returns an error when called with
// a [dbmodel.Lease] which has a nil [dbmodel.Daemon].
func TestConvertLeaseFromRestAPIWithNilDaemon(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 - 1,
		},
	}
	result, err := convertLeaseToRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Daemon")
}

// Verify that [convertLeaseToRestAPI] returns an error when called with
// a [dbmodel.Lease] which has a nil [dbmodel.Subnet].
func TestConvertLeaseFromRestAPIWithNilSubnet(t *testing.T) {
	lease := dbmodel.Lease{
		Daemon: &dbmodel.Daemon{
			ID:   1,
			Name: daemonname.DHCPv4,
		},
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 - 1,
		},
	}
	result, err := convertLeaseToRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Subnet")
}

// Verify that [convertLeaseToRestAPI] correctly converts a [dbmodel.Lease]
// to a [dhcp.Lease] when provided with complete and valid input.
func TestConvertLeaseFromRestAPIWithValidLease(t *testing.T) {
	duid := "00:00:00:00:00:00:00:00:01:01"
	lease := dbmodel.Lease{
		DaemonID: 1,
		SubnetID: 9,
		Daemon: &dbmodel.Daemon{
			ID:   1,
			Name: daemonname.DHCPv6,
		},
		Lease: keadata.Lease{
			State:         1,
			CLTT:          1776459817,
			IPAddress:     "fe80::9",
			PrefixLength:  128,
			DUID:          keadata.NewColonSeparatedHexStr(&duid),
			ValidLifetime: 3600,
		},
		Subnet: &dbmodel.Subnet{
			ID:     9,
			Prefix: "fe80::/64",
		},
	}
	result, err := convertLeaseToRestAPI(&lease)

	require.Nil(t, err)
	require.NotNil(t, result)
	require.EqualValues(t, lease.CLTT, *result.Cltt)
}

// Verify that [convertSortFieldToColumnName] converts all of the supported sort
// fields into the correct column names.
func TestConvertSortFieldToColumnNameHandlesAllCases(t *testing.T) {
	testCases := []struct {
		description     string
		sortField       string
		expectedColName dbmodel.GetLeasesByPageSortColumnName
	}{
		{
			description:     "HW Address",
			sortField:       string(models.LeaseListSortFieldHwAddress),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameHwAddress,
		},
		{
			description:     "IP Address",
			sortField:       string(models.LeaseListSortFieldIPAddress),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameIPAddress,
		},
		{
			description:     "Hostname",
			sortField:       string(models.LeaseListSortFieldHostname),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameHostname,
		},
		{
			description:     "Client ID",
			sortField:       string(models.LeaseListSortFieldClientID),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameClientID,
		},
		{
			description:     "DUID",
			sortField:       string(models.LeaseListSortFieldDuid),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameDuid,
		},
		{
			description:     "CLTT",
			sortField:       string(models.LeaseListSortFieldCltt),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameCltt,
		},
		{
			description:     "Valid Lifetime",
			sortField:       string(models.LeaseListSortFieldValidLifetime),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameValidLifetime,
		},
		{
			description:     "PrefixLength",
			sortField:       string(models.LeaseListSortFieldPrefixLength),
			expectedColName: dbmodel.GetLeasesByPageSortColumnNamePrefixLength,
		},
		{
			description:     "Unknown field",
			sortField:       "potato",
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameNone,
		},
		{
			description:     "Empty string",
			sortField:       "",
			expectedColName: dbmodel.GetLeasesByPageSortColumnNameNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := convertSortFieldToColumnName(tc.sortField)
			require.Equal(t, tc.expectedColName, result)
		})
	}
}

// Verify that [getLeases] propagates errors from [GetLeasesByPage] to its caller.
func TestGetLeasesPropagatesErrorFromGetLeasesByPage(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(dbSettings, db, fec)
	require.NoError(t, err)

	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Act
	filters := dbmodel.LeasesByPageFilters{}
	db.Close()
	leases, err := rapi.getLeases(0, 10, filters, "", dbmodel.SortDirAsc)

	// Assert
	require.Nil(t, leases)
	require.ErrorContains(t, err, "database is closed")
}

// testHelperMakeUser is a test helper function which adds a user to the
// database with a given username and password, and ensures that the operation
// succeeded.
func testHelperMakeUser(t *testing.T, db *dbops.PgDB, user *dbmodel.SystemUser, password string) {
	t.Helper()
	con, err := dbmodel.CreateUserWithPassword(db, user, password)
	require.NoError(t, err)
	require.False(t, con)
}

// Verify that [GetLeaseList] enforces user authentication properly:
// - Logged-out users cannot see leases.
// - Read only users cannot see leases.
// - Admin users can see leases.
// - Super Admin users can see leases.
func TestGetLeaseListUserAuth(t *testing.T) {
	roUser := &dbmodel.SystemUser{
		Email:    "san.zhang@example.com",
		Lastname: "张",
		Name:     "三",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.ReadOnlyGroupID},
		},
	}
	adminUser := &dbmodel.SystemUser{
		Email:    "fulana.alfulaniyya@example.com",
		Lastname: "AlFulaniyya",
		Name:     "Fulan",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.AdminGroupID},
		},
	}
	superAdminUser := &dbmodel.SystemUser{
		Email:    "erika.mustermann@example.com",
		Lastname: "Mustermann",
		Name:     "Erika",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.SuperAdminGroupID},
		},
	}
	t.Run("deny logged-out user", func(t *testing.T) {
		db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
		defer teardown()

		ctx := context.Background()
		fec := &storktest.FakeEventCenter{}
		rapi, err := NewRestAPI(dbSettings, db, fec)
		require.NoError(t, err)

		ctx, err = rapi.SessionManager.Load(ctx, "")
		require.NoError(t, err)

		getLeaseListParams := dhcp.GetLeaseListParams{}
		rsp := rapi.GetLeaseList(ctx, getLeaseListParams)
		require.IsType(t, &dhcp.GetLeaseListDefault{}, rsp)
	})
	t.Run("deny read-only user", func(t *testing.T) {
		db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
		defer teardown()

		ctx := context.Background()
		fec := &storktest.FakeEventCenter{}
		rapi, err := NewRestAPI(dbSettings, db, fec)
		require.NoError(t, err)

		ctx, err = rapi.SessionManager.Load(ctx, "")
		require.NoError(t, err)

		// Create testing users in the database.
		testHelperMakeUser(t, db, roUser, "pass")

		err = rapi.SessionManager.LoginHandler(ctx, roUser)
		require.NoError(t, err)

		getLeaseListParams := dhcp.GetLeaseListParams{}
		rsp := rapi.GetLeaseList(ctx, getLeaseListParams)
		require.IsType(t, &dhcp.GetLeaseListDefault{}, rsp)
	})
	t.Run("allow admin user", func(t *testing.T) {
		db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
		defer teardown()

		ctx := context.Background()
		fec := &storktest.FakeEventCenter{}
		rapi, err := NewRestAPI(dbSettings, db, fec)
		require.NoError(t, err)

		ctx, err = rapi.SessionManager.Load(ctx, "")
		require.NoError(t, err)

		// Create testing users in the database.
		testHelperMakeUser(t, db, adminUser, "pass1")

		err = rapi.SessionManager.LoginHandler(ctx, adminUser)
		require.NoError(t, err)

		getLeaseListParams := dhcp.GetLeaseListParams{}
		rsp := rapi.GetLeaseList(ctx, getLeaseListParams)
		log.WithField("rsp", rsp).Info("response structure")
		require.IsType(t, &dhcp.GetLeaseListOK{}, rsp)
	})
	t.Run("allow super-admin user", func(t *testing.T) {
		db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
		defer teardown()

		ctx := context.Background()
		fec := &storktest.FakeEventCenter{}
		rapi, err := NewRestAPI(dbSettings, db, fec)
		require.NoError(t, err)

		ctx, err = rapi.SessionManager.Load(ctx, "")
		require.NoError(t, err)

		// Create testing users in the database.
		testHelperMakeUser(t, db, superAdminUser, "pass2")

		err = rapi.SessionManager.LoginHandler(ctx, superAdminUser)
		require.NoError(t, err)

		getLeaseListParams := dhcp.GetLeaseListParams{}
		rsp := rapi.GetLeaseList(ctx, getLeaseListParams)
		require.IsType(t, &dhcp.GetLeaseListOK{}, rsp)
	})
}

// Verify that [GetLeaseList] propagates an error from [getLeases].
func TestGetLeaseListPropagatesErrorFromGetLeases(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(dbSettings, db, fec)
	require.NoError(t, err)

	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	superAdminUser := &dbmodel.SystemUser{
		Email:    "erika.mustermann@example.com",
		Lastname: "Mustermann",
		Name:     "Erika",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.SuperAdminGroupID},
		},
	}
	testHelperMakeUser(t, db, superAdminUser, "pass2")

	err = rapi.SessionManager.LoginHandler(ctx, superAdminUser)
	require.NoError(t, err)

	getLeaseListParams := dhcp.GetLeaseListParams{}
	db.Close()
	rsp := rapi.GetLeaseList(ctx, getLeaseListParams)
	require.IsType(t, &dhcp.GetLeaseListDefault{}, rsp)
}

// helperSetUpLeases sets up two leases (and the associated supporting data) for the
// following test.
func helperSetUpLeases(t *testing.T, db *dbops.PgDB) ([]*dbmodel.Lease, *dbmodel.Machine) {
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	accessPoint := []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "cool.example.org",
			Port:    int64(1234),
			Key:     "",
		},
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv6, true, accessPoint)
	configStr := `{
		"Dhcp6": {
			"subnet6": [
				{ "id": 123, "subnet": "2001:db8:1::/64" }
			]
		}
	}`
	config, err := keaconfig.NewConfig([]byte(configStr))
	require.NoError(t, err)
	daemon.KeaDaemon.Config = &dbmodel.KeaConfig{
		Config: config,
	}
	daemon.KeaDaemon.Config.DHCPv6Config.LeaseDatabase = &keaconfig.Database{
		Type: "memfile",
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	subnet := &dbmodel.Subnet{
		Prefix: "2001:db8:1::/64",
		LocalSubnets: []*dbmodel.LocalSubnet{
			{
				DaemonID:      daemon.ID,
				LocalSubnetID: 123,
				AddressPools: []dbmodel.AddressPool{
					{
						LowerBound: "2001:db8:1::1",
						UpperBound: "2001:db8:1::ffff",
					},
				},
			},
		},
	}
	err = dbmodel.AddSubnet(db, subnet)
	require.NoError(t, err)

	duid0through7 := "00:01:02:03:04:05:06:07"
	duid1through8 := "01:02:03:04:05:06:07:08"
	leases := []*dbmodel.Lease{
		{
			DaemonID: daemon.ID,
			SubnetID: subnet.ID,
			Lease: keadata.Lease{
				Family:        6,
				DUID:          keadata.NewColonSeparatedHexStr(&duid0through7),
				IPAddress:     "2001:db8:1::404",
				CLTT:          10002,
				Hostname:      "client.example",
				State:         keadata.LeaseStateDefault,
				ValidLifetime: 3600,
				LocalSubnetID: 123,
			},
		},
		{
			DaemonID: daemon.ID,
			SubnetID: subnet.ID,
			Lease: keadata.Lease{
				Family:        6,
				DUID:          keadata.NewColonSeparatedHexStr(&duid1through8),
				IPAddress:     "2001:db8:1::408",
				CLTT:          10002,
				Hostname:      "client.example",
				State:         keadata.LeaseStateDefault,
				ValidLifetime: 3600,
				LocalSubnetID: 123,
			},
		},
	}

	for _, lease := range leases {
		err := dbmodel.AddLease(db, lease)
		require.NoError(t, err)
		require.NotZero(t, lease.ID)
	}

	return leases, machine
}

// Verify that [GetLeaseList] handles a request with every parameter set.
func TestGetLeaseListHandlesParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	leases, machine := helperSetUpLeases(t, db)
	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(dbSettings, db, fec)
	require.NoError(t, err)

	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	superAdminUser := &dbmodel.SystemUser{
		Email:    "erika.mustermann@example.com",
		Lastname: "Mustermann",
		Name:     "Erika",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.SuperAdminGroupID},
		},
	}
	testHelperMakeUser(t, db, superAdminUser, "pass2")

	err = rapi.SessionManager.LoginHandler(ctx, superAdminUser)
	require.NoError(t, err)

	start := int64(0)
	limit := int64(3)
	sortField := string(models.LeaseListSortFieldIPAddress)
	sortDir := string(dbmodel.SortDirDesc)
	daemonID := leases[0].DaemonID
	machineID := machine.ID
	subnetID := leases[0].SubnetID
	localSubnetID := int64(leases[0].LocalSubnetID)
	filterText := "client.example"

	// Act
	getLeaseListParams := dhcp.GetLeaseListParams{
		Start:         &start,
		Limit:         &limit,
		SortField:     &sortField,
		SortDir:       &sortDir,
		DaemonID:      &daemonID,
		MachineID:     &machineID,
		SubnetID:      &subnetID,
		LocalSubnetID: &localSubnetID,
		Text:          &filterText,
	}
	rsp := rapi.GetLeaseList(ctx, getLeaseListParams)

	// Assert
	require.IsType(t, &dhcp.GetLeaseListOK{}, rsp)
	require.NotNil(t, rsp)
	okRsp := rsp.(*dhcp.GetLeaseListOK)
	require.NotNil(t, okRsp.Payload)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.NotNil(t, okRsp.Payload.Items)
	require.Len(t, okRsp.Payload.Items, 2)
	require.NotNil(t, okRsp.Payload.Items[0].IPAddress)
	require.Equal(t, leases[1].IPAddress, *okRsp.Payload.Items[0].IPAddress)
	require.NotNil(t, okRsp.Payload.Items[1].IPAddress)
	require.Equal(t, leases[0].IPAddress, *okRsp.Payload.Items[1].IPAddress)
}
