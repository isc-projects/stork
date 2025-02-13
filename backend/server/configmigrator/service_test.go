package configmigrator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Creates a migration service with some predefined migrations.
func newTestService() Service {
	service := NewService().(*service)
	// Migration in progress.
	service.migrations["in-progress"] = &migration{
		id:             "in-progress",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   nil,
		processedItems: 10,
		totalItems:     100,
		errors:         make(map[int64]error),
		cancelDate:     time.Time{},
		entityType:     EntityTypeHost,
		cancelFunc: func() {
			service.migrations["in-progress"].cancelDate = time.Date(
				2025, 2, 1, 13, 0, 0, 0, time.UTC,
			)
		},
	}
	// Finished migration.
	service.migrations["finished"] = &migration{
		id:             "finished",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 2, 11, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 2, 12, 0, 0, 0, time.UTC),
		generalError:   nil,
		processedItems: 100,
		totalItems:     100,
		errors:         map[int64]error{42: errors.New("error")},
		cancelDate:     time.Time{},
		entityType:     EntityTypeHost,
	}
	// Canceled migration.
	service.migrations["canceled"] = &migration{
		id:             "canceled",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 3, 11, 0, 0, 0, time.UTC),
		cancelDate:     time.Date(2025, 2, 3, 12, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 3, 13, 0, 0, 0, time.UTC),
		generalError:   errors.New("canceled"),
		processedItems: 10,
		totalItems:     100,
		errors:         make(map[int64]error),
		entityType:     EntityTypeHost,
	}
	// Canceling migration.
	service.migrations["canceling"] = &migration{
		id:             "canceling",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 4, 11, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   nil,
		cancelDate:     time.Date(2025, 2, 4, 12, 0, 0, 0, time.UTC),
		processedItems: 10,
		totalItems:     100,
		errors:         make(map[int64]error),
		entityType:     EntityTypeHost,
	}
	// General error occurred.
	service.migrations["general-error"] = &migration{
		id:             "general-error",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 5, 11, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 5, 12, 0, 0, 0, time.UTC),
		generalError:   errors.New("general error"),
		processedItems: 10,
		totalItems:     100,
		errors:         make(map[int64]error),
		cancelDate:     time.Time{},
		entityType:     EntityTypeHost,
	}
	return service
}

// Test that the service instance is created correctly.
func TestNewService(t *testing.T) {
	// Arrange & Act
	service := NewService().(*service)

	// Assert
	require.NotNil(t, service)
	require.NotNil(t, service.migrations)
}

// Test that the migrations are listed correctly regardless their state.
func TestGetMigrations(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migrations := service.GetMigrations()

	// Assert
	require.Len(t, migrations, 5)
	require.EqualValues(t, "in-progress", migrations[0].ID)
	require.EqualValues(t, "finished", migrations[1].ID)
	require.EqualValues(t, "canceled", migrations[2].ID)
	require.EqualValues(t, "canceling", migrations[3].ID)
	require.EqualValues(t, "general-error", migrations[4].ID)
}

// Test that the migration is retrieved correctly.
func TestGetMigration(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migrationInProgress, okInProgress := service.GetMigration("in-progress")
	migrationFinished, okFinished := service.GetMigration("finished")
	migrationCanceled, okCanceled := service.GetMigration("canceled")
	migrationCanceling, okCanceling := service.GetMigration("canceling")
	migrationGeneralError, okGeneralError := service.GetMigration("general-error")
	migrationUnknown, okUnknown := service.GetMigration("unknown")

	// Assert
	require.True(t, okInProgress)
	require.EqualValues(t, "in-progress", migrationInProgress.ID)
	require.True(t, okFinished)
	require.EqualValues(t, "finished", migrationFinished.ID)
	require.True(t, okCanceled)
	require.EqualValues(t, "canceled", migrationCanceled.ID)
	require.True(t, okCanceling)
	require.EqualValues(t, "canceling", migrationCanceling.ID)
	require.True(t, okGeneralError)
	require.EqualValues(t, "general-error", migrationGeneralError.ID)
	require.False(t, okUnknown)
	require.EqualValues(t, MigrationStatus{}, migrationUnknown)
}

// Test that the executing of the stop migration function calls the cancel
// function.
func TestStopMigrationCallsCancelFunction(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("in-progress")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the migration that is unknown cannot be stopped.
func TestStopUnknownMigration(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migration, ok := service.StopMigration("unknown")

	// Assert
	require.False(t, ok)
	require.EqualValues(t, MigrationStatus{}, migration)
}

// Test that the finished migration cannot be stopped again.
func TestStopFinishedMigration(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("finished")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.False(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.NotZero(t, status.EndDate)
}

// Test that the canceling migration cannot be stopped again.
func TestStopCancelingMigration(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("canceling")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the finished migrations can be cleared.
func TestClearFinishedMigrations(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	service.ClearFinishedMigrations()

	// Assert
	migrations := service.GetMigrations()
	require.Len(t, migrations, 2)
	require.EqualValues(t, "in-progress", migrations[0].ID)
	require.EqualValues(t, "canceling", migrations[1].ID)
}
