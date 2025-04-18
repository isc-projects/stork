package configmigrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -package=configmigrator -destination=migratormock_test.go isc.org/stork/server/configmigrator Migrator

// A helper function to read all items from the channels.
func readChannels(ch <-chan migrationChunk) (errs []MigrationError, generalErr error) {
	errs = make([]MigrationError, 0)

	for chunk := range ch {
		generalErr = chunk.generalErr
		errs = append(errs, chunk.errs...)
	}

	return
}

// Test that runner doesn't crash if there are no items to migrate.
func TestRunMigrationEmpty(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	gomock.InOrder(
		mock.EXPECT().Begin(),
		mock.EXPECT().LoadItems(int64(0)).Return(int64(0), nil),
		mock.EXPECT().End(),
	)

	ctx := context.Background()

	// Act
	chunks := runMigration(ctx, mock)
	allChunks, err := readChannels(chunks)

	// Assert
	require.NoError(t, err)
	require.Empty(t, allChunks)
}

// Test that runner migrates all items.
func TestRunMigration(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)

	gomock.InOrder(
		mock.EXPECT().Begin(),
		mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), nil),
		mock.EXPECT().Migrate().Return([]MigrationError{}),
		mock.EXPECT().LoadItems(gomock.Eq(int64(10))).Return(int64(10), nil),
		mock.EXPECT().Migrate().Return([]MigrationError{}),
		mock.EXPECT().LoadItems(gomock.Eq(int64(20))).Return(int64(5), nil),
		mock.EXPECT().Migrate().Return([]MigrationError{}),
		mock.EXPECT().LoadItems(gomock.Eq(int64(25))).Return(int64(0), nil),
		mock.EXPECT().End(),
	)

	ctx := context.Background()

	// Act
	chunks := runMigration(ctx, mock)
	errs, err := readChannels(chunks)

	// Assert
	require.NoError(t, err)
	require.Empty(t, errs) // updated to check allChunks instead of errs
}

// Test that the migration is interrupted if any error occurs in the initial
// phase of the migration.
func TestRunMigrateInterruptOnBeginError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)

	mock.EXPECT().Begin().Return(errors.New("begin error"))

	// Act
	chunks := runMigration(context.Background(), mock)
	errs, err := readChannels(chunks)

	// Assert
	require.ErrorContains(t, err, "begin error")
	require.Empty(t, errs)
}

// Test that the migration done channel doesn't contain an error occurred
// during the cleanup.
func TestRunMigrationCleanupError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	gomock.InOrder(
		mock.EXPECT().Begin(),
		mock.EXPECT().LoadItems(int64(0)).Return(int64(0), nil),
		mock.EXPECT().End().Return(errors.New("cleanup error")),
	)

	ctx := context.Background()

	// Act
	chunks := runMigration(ctx, mock)
	allChunks, err := readChannels(chunks)

	// Assert
	require.NoError(t, err)
	require.Empty(t, allChunks)
}

// Test that the migration errors are aggregated.
func TestRunMigrationAggregatesErrors(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	mock.EXPECT().Begin()
	mock.EXPECT().End()
	mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(10))).Return(int64(5), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(15))).Return(int64(0), nil)

	callCount := 0
	mock.EXPECT().Migrate().DoAndReturn(func() []MigrationError {
		callCount++
		return []MigrationError{
			{
				ID:    int64(callCount),
				Error: errors.Errorf("error %d", callCount),
				Type:  EntityTypeHost,
				Label: fmt.Sprintf("host %d", callCount),
			},
		}
	}).Times(2)

	ctx := context.Background()

	// Act
	chunks := runMigration(ctx, mock)
	errs, err := readChannels(chunks)

	// Assert
	require.NoError(t, err)
	require.Len(t, errs, 2)
	require.EqualValues(t, 1, errs[0].ID)
	require.EqualError(t, errs[0].Error, "error 1")
	require.Equal(t, EntityTypeHost, errs[0].Type)
	require.Equal(t, "host 1", errs[0].Label)
	require.EqualValues(t, 2, errs[1].ID)
	require.EqualError(t, errs[1].Error, "error 2")
	require.Equal(t, "host 2", errs[1].Label)
	require.Equal(t, EntityTypeHost, errs[1].Type)
}

// Test that the runner interrupts the migration after the loading error.
func TestRunMigrationInterruptOnLoadingError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	gomock.InOrder(
		mock.EXPECT().Begin(),
		mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), errors.New("loading error")),
		mock.EXPECT().End(),
	)

	ctx := context.Background()

	// Act
	chunks := runMigration(ctx, mock)
	errs, err := readChannels(chunks)

	// Assert
	require.ErrorContains(t, err, "loading error")
	require.Empty(t, errs)
}

// Test that the migration may be canceled.
func TestRunMigrationCanceled(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	gomock.InOrder(
		mock.EXPECT().Begin(),
		mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), nil),
		mock.EXPECT().Migrate().Return([]MigrationError{}),
		mock.EXPECT().LoadItems(gomock.Eq(int64(10))).Return(int64(10), nil),
		mock.EXPECT().Migrate().Do(func() {
			cancel()
		}).Return([]MigrationError{}),
		mock.EXPECT().End(),
	)

	// Act
	chunks := runMigration(ctx, mock)
	errs, err := readChannels(chunks)
	cancel()

	// Assert
	require.ErrorContains(t, err, "context canceled")
	require.Empty(t, errs)
}
