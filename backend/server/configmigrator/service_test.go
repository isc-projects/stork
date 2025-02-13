package configmigrator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

// Test that the migration is started and executed asynchronously.
func TestStartAndExecuteMigration(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(250), nil)
	migrator.EXPECT().GetEntityType().Return(EntityTypeHost)

	migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(200))).Return(int64(50), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(250))).Return(int64(0), nil)

	// Blocks the migration runner until the assertion of a particular chunk is
	// finished.
	assertionFinishedChan := make(chan struct{})
	defer close(assertionFinishedChan)

	migrator.EXPECT().Migrate().Do(func() {
		// The migrator will wait for the assertion to finish before it
		// continues. Also, the assertion will wait for the migrator to finish
		// migrating the chunk before it continues.
		<-assertionFinishedChan
		// The runner does additional processing after the migrator does its
		// job. So, we cannot immediately run the assertions after the channel
		// is empty. We need to wait for the runner to finish its job by
		// calling t.Eventually.
	}).Return(map[int64]error{}).Times(3)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	// Act
	initialStatus, err := service.StartMigration(ctx, migrator)

	// Assert
	require.NoError(t, err)

	// Check the initial status.
	require.NotEmpty(t, initialStatus.ID)
	require.False(t, initialStatus.Canceling)
	require.Equal(t, "value", initialStatus.Context.Value(contextKey("key")))
	require.Zero(t, initialStatus.EndDate)
	require.Zero(t, initialStatus.GeneralError)
	require.Zero(t, initialStatus.Progress)
	require.Zero(t, initialStatus.EstimatedLeftTime)
	require.Empty(t, initialStatus.Errors)
	require.Equal(t, EntityTypeHost, initialStatus.EntityType)

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-100.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, firstChunkStatus.ID)
	require.False(t, firstChunkStatus.Canceling)
	require.Equal(t, "value", firstChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, firstChunkStatus.EndDate)
	require.Zero(t, firstChunkStatus.GeneralError)
	require.InDelta(t, firstChunkStatus.Progress, 100.0/250.0, 1e-6)
	require.NotZero(t, firstChunkStatus.EstimatedLeftTime)
	require.NotZero(t, firstChunkStatus.ElapsedTime)
	require.Empty(t, firstChunkStatus.Errors)
	require.Equal(t, initialStatus.EntityType, firstChunkStatus.EntityType)

	// Wait for the second chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-200.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, secondChunkStatus.ID)
	require.False(t, secondChunkStatus.Canceling)
	require.Equal(t, "value", secondChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, secondChunkStatus.EndDate)
	require.Zero(t, secondChunkStatus.GeneralError)
	require.InDelta(t, secondChunkStatus.Progress, 200.0/250.0, 1e-6)
	require.NotZero(t, secondChunkStatus.EstimatedLeftTime)
	require.NotZero(t, secondChunkStatus.ElapsedTime)
	require.Empty(t, secondChunkStatus.Errors)
	require.Equal(t, initialStatus.EntityType, secondChunkStatus.EntityType)

	// Wait for the third chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-1.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the third chunk is migrated.
	thirdChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, thirdChunkStatus.ID)
	require.False(t, thirdChunkStatus.Canceling)
	require.Equal(t, "value", thirdChunkStatus.Context.Value(contextKey("key")))
	require.NotZero(t, thirdChunkStatus.EndDate)
	require.Zero(t, thirdChunkStatus.GeneralError)
	require.InDelta(t, thirdChunkStatus.Progress, 1.0, 1e-6)
	require.Zero(t, thirdChunkStatus.EstimatedLeftTime)
	require.NotZero(t, thirdChunkStatus.ElapsedTime)
	require.Empty(t, thirdChunkStatus.Errors)
	require.Equal(t, initialStatus.EntityType, thirdChunkStatus.EntityType)
}

// Test that the migration errors are aggregated.

// Test that the migration is not started if an error occurs in the initial
// phase.

// Test that the migration has unique ID even if some migrations are started
// in exactly the same time.

// Test that the loading error interrupts the migration.

// Test that the migration can be canceled.

// Test that the migration status doesn't contain the cancellation stuff.
