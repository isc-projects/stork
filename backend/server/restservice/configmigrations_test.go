package restservice

import (
	context "context"
	http "net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"isc.org/stork/server/configmigrator"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
)

// Test that the all migrations can be returned regardless of their status.
func TestGetMigrations(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	ctxAuthor, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	migrationErred := &configmigrator.MigrationStatus{
		ID:                  12341,
		Context:             ctxAuthor,
		StartDate:           time.Date(2025, 2, 13, 10, 24, 45, 432000000, time.UTC),
		EndDate:             time.Time{},
		Canceling:           false,
		ProcessedItemsCount: 2,
		TotalItemsCount:     10,
		Errors: []configmigrator.MigrationError{
			{Error: errors.New("foo"), ID: 4, Label: "host-4", Type: configmigrator.EntityTypeHost},
			{Error: errors.New("bar"), ID: 2, Label: "host-2", Type: configmigrator.EntityTypeHost},
		},
		GeneralError:      nil,
		ElapsedTime:       5 * time.Second,
		EstimatedLeftTime: 1 * time.Minute,
	}

	migrationInProgress := &configmigrator.MigrationStatus{
		ID:                  12342,
		Context:             ctxAuthor,
		StartDate:           time.Date(2025, 2, 14, 11, 25, 46, 432000000, time.UTC),
		EndDate:             time.Time{},
		Canceling:           false,
		ProcessedItemsCount: 5,
		TotalItemsCount:     10,
		Errors:              nil,
		GeneralError:        nil,
		ElapsedTime:         10 * time.Second,
		EstimatedLeftTime:   2 * time.Minute,
	}

	migrationFinished := &configmigrator.MigrationStatus{
		ID:                  12343,
		Context:             ctxAuthor,
		StartDate:           time.Date(2025, 2, 15, 12, 26, 47, 432000000, time.UTC),
		EndDate:             time.Date(2025, 2, 15, 12, 27, 48, 432000000, time.UTC),
		Canceling:           false,
		ProcessedItemsCount: 10,
		TotalItemsCount:     10,
		Errors:              nil,
		GeneralError:        nil,
		ElapsedTime:         15 * time.Second,
		EstimatedLeftTime:   0 * time.Minute,
	}

	migrationService.EXPECT().GetMigrations().Return(
		[]*configmigrator.MigrationStatus{
			migrationErred, migrationInProgress, migrationFinished,
		},
	)

	// Act
	rsp := rapi.GetMigrations(context.Background(), dhcp.GetMigrationsParams{})

	// Assert
	require.IsType(t, &dhcp.GetMigrationsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetMigrationsOK)

	require.Len(t, okRsp.Payload.Items, 3)

	status := okRsp.Payload.Items[0]
	require.EqualValues(t, 12341, status.ID)
	require.Equal(t, "2025-02-13T10:24:45.432Z", status.StartDate.String())
	require.EqualValues(t, 2, status.ProcessedItemsCount)
	require.EqualValues(t, 2, status.Errors.Total)
	require.Len(t, status.Errors.Items, 2)
	require.ElementsMatch(t, []*models.MigrationError{
		{Error: "foo", ID: 4, Label: "host-4", Type: "host"},
		{Error: "bar", ID: 2, Label: "host-2", Type: "host"},
	}, status.Errors.Items)

	status = okRsp.Payload.Items[1]
	require.EqualValues(t, 12342, status.ID)
	require.Equal(t, "2025-02-14T11:25:46.432Z", status.StartDate.String())
	require.EqualValues(t, 5, status.ProcessedItemsCount)
	require.EqualValues(t, 0, status.Errors.Total)
	require.Len(t, status.Errors.Items, 0)

	status = okRsp.Payload.Items[2]
	require.EqualValues(t, 12343, status.ID)
	require.Equal(t, "2025-02-15T12:26:47.432Z", status.StartDate.String())
	require.EqualValues(t, 10, status.ProcessedItemsCount)
	require.EqualValues(t, int64(0), status.Errors.Total)
	require.Len(t, status.Errors.Items, 0)
}

// Test that the endpoint correctly triggers the deletion of all finished
// migrations.
func TestDeleteFinishedMigrations(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	migrationService.EXPECT().ClearFinishedMigrations()

	// Act
	rsp := rapi.DeleteFinishedMigrations(
		context.Background(),
		dhcp.DeleteFinishedMigrationsParams{},
	)

	// Assert
	require.IsType(t, &dhcp.DeleteFinishedMigrationsOK{}, rsp)
}

// Test that the HTTP 404 status is returned when trying to get a migration
// that does not exist.
func TestGetMigrationNotFound(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	migrationService.EXPECT().GetMigration(configmigrator.MigrationIdentifier(12341)).Return(nil, false)

	// Act
	rsp := rapi.GetMigration(context.Background(), dhcp.GetMigrationParams{ID: 12341})

	// Assert
	require.IsType(t, &dhcp.GetMigrationDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.GetMigrationDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}

// Test that the particular migration is returned correctly if it exists.
func TestGetMigration(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	migrationStatus := &configmigrator.MigrationStatus{
		ID:                  12341,
		Context:             ctx,
		StartDate:           time.Date(2025, 2, 13, 10, 24, 45, 432000000, time.UTC),
		EndDate:             time.Time{},
		Canceling:           false,
		ProcessedItemsCount: 2,
		TotalItemsCount:     10,
		Errors: []configmigrator.MigrationError{
			{Error: errors.New("foo"), ID: 4, Label: "host-4", Type: configmigrator.EntityTypeHost},
			{Error: errors.New("bar"), ID: 2, Label: "host-2", Type: configmigrator.EntityTypeHost},
		},
		GeneralError:      nil,
		ElapsedTime:       5 * time.Second,
		EstimatedLeftTime: 1 * time.Minute,
	}

	migrationService.EXPECT().GetMigration(configmigrator.MigrationIdentifier(12341)).Return(migrationStatus, true)

	// Act
	rsp := rapi.GetMigration(context.Background(), dhcp.GetMigrationParams{ID: 12341})

	// Assert
	require.IsType(t, &dhcp.GetMigrationOK{}, rsp)
	okRsp := rsp.(*dhcp.GetMigrationOK)

	require.EqualValues(t, 12341, okRsp.Payload.ID)
	require.Equal(t, "2025-02-13T10:24:45.432Z", okRsp.Payload.StartDate.String())
	require.EqualValues(t, 2, okRsp.Payload.ProcessedItemsCount)
	require.EqualValues(t, 10, okRsp.Payload.TotalItemsCount)
	require.EqualValues(t, 2, okRsp.Payload.Errors.Total)
	require.Len(t, okRsp.Payload.Errors.Items, 2)
	require.ElementsMatch(t, []*models.MigrationError{
		{Error: "foo", ID: 4, Label: "host-4", Type: "host"},
		{Error: "bar", ID: 2, Label: "host-2", Type: "host"},
	}, okRsp.Payload.Errors.Items)
}

// Test that the cancellation endpoint correctly triggers the cancellation of
// the migration.
func TestPutMigration(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	migrationStatus := &configmigrator.MigrationStatus{
		ID:                  12341,
		Context:             ctx,
		StartDate:           time.Date(2025, 2, 13, 10, 24, 45, 432000000, time.UTC),
		EndDate:             time.Time{},
		Canceling:           true,
		ProcessedItemsCount: 2,
		TotalItemsCount:     10,
		Errors: []configmigrator.MigrationError{
			{Error: errors.New("foo"), ID: 4, Label: "host-4", Type: configmigrator.EntityTypeHost},
			{Error: errors.New("bar"), ID: 2, Label: "host-2", Type: configmigrator.EntityTypeHost},
		},
		GeneralError:      nil,
		ElapsedTime:       5 * time.Second,
		EstimatedLeftTime: 1 * time.Minute,
	}

	migrationService.EXPECT().
		StopMigration(configmigrator.MigrationIdentifier(12341)).
		Return(migrationStatus, true)

	// Act
	rsp := rapi.PutMigration(
		context.Background(),
		dhcp.PutMigrationParams{ID: 12341},
	)

	// Assert
	require.IsType(t, &dhcp.PutMigrationOK{}, rsp)
	okRsp := rsp.(*dhcp.PutMigrationOK)

	require.EqualValues(t, 12341, okRsp.Payload.ID)
	require.Equal(t, "2025-02-13T10:24:45.432Z", okRsp.Payload.StartDate.String())
	require.EqualValues(t, 2, okRsp.Payload.ProcessedItemsCount)
	require.EqualValues(t, 10, okRsp.Payload.TotalItemsCount)
	require.EqualValues(t, 2, okRsp.Payload.Errors.Total)
	require.Len(t, okRsp.Payload.Errors.Items, 2)
	require.ElementsMatch(t, []*models.MigrationError{
		{Error: "foo", ID: 4, Label: "host-4", Type: "host"},
		{Error: "bar", ID: 2, Label: "host-2", Type: "host"},
	}, okRsp.Payload.Errors.Items)
	require.True(t, okRsp.Payload.Canceling)
}

// Test that the cancellation endpoint returns a HTTP 404 Not Found status when
// trying to cancel a migration that does not exist.
func TestPutMigrationNotFound(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migrationService := NewMockMigrationManager(ctrl)

	rapi, err := NewRestAPI(dbSettings, db, migrationService)
	require.NoError(t, err)

	migrationService.EXPECT().
		StopMigration(configmigrator.MigrationIdentifier(12341)).
		Return(nil, false)

	// Act
	rsp := rapi.PutMigration(
		context.Background(),
		dhcp.PutMigrationParams{ID: 12341},
	)

	// Assert
	require.IsType(t, &dhcp.PutMigrationDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.PutMigrationDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}
