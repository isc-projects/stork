package restservice

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test/dbmodel"
	log "github.com/sirupsen/logrus"
)

func TestConvertLeaseFromRestAPIWithNilLease(t *testing.T) {
	result, err := convertLeaseFromRestAPI(nil)

	require.Nil(t, result)
	require.ErrorContains(t, err, "nil")
}

func TestConvertLeaseFromRestAPIWithCLTTTooBig(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 + 1,
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "CLTT")
}

func TestConvertLeaseFromRestAPIWithNilDaemon(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 - 1,
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Daemon")
}

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
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Subnet")
}

func TestConvertLeaseFromRestAPIWithValidLease(t *testing.T) {
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
			DUID:          "00:00:00:00:00:00:00:00:01:01",
			ValidLifetime: 3600,
		},
		Subnet: &dbmodel.Subnet{
			ID:     9,
			Prefix: "fe80::/64",
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, err)
	require.NotNil(t, result)
	require.EqualValues(t, lease.CLTT, *result.Cltt)
}

func testHelperMakeUser(t *testing.T, db *dbops.PgDB, user *dbmodel.SystemUser, password string) {
	t.Helper()
	con, err := dbmodel.CreateUserWithPassword(db, user, password)
	require.NoError(t, err)
	require.False(t, con)
}

func TestGetLeaseListUserAuth(t *testing.T) {
	roUser := &dbmodel.SystemUser{
		Email: "san.zhang@example.com",
		Lastname: "张",
		Name: "三",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.ReadOnlyGroupID},
		},
	}
	adminUser := &dbmodel.SystemUser{
		Email: "fulana.alfulaniyya@example.com",
		Lastname: "AlFulaniyya",
		Name: "Fulan",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.AdminGroupID},
		},
	}
	superAdminUser := &dbmodel.SystemUser{
		Email: "erika.mustermann@example.com",
		Lastname: "Mustermann",
		Name: "Erika",
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

func TestGetLeaseList(t *testing.T) {
	require.Nil(t, nil)
}
