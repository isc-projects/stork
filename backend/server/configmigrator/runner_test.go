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
func readChannels(ch <-chan migrationChunk, done <-chan error) (errs []MigrationError, err error) {
	errs = make([]MigrationError, 0)

	for {
		select {
		case err = <-done:
			return
		case chunk := <-ch:
			errs = append(errs, chunk.errs...)
		}
	}
}

// Test that runner doesn't crash if there are no items to migrate.
func TestRunMigrationEmpty(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	mock.EXPECT().LoadItems(int64(0)).Return(int64(0), nil)

	ctx := context.Background()

	// Act
	chunks, done := runMigration(ctx, mock)
	allChunks, err := readChannels(chunks, done)

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
	mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(10))).Return(int64(10), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(20))).Return(int64(5), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(25))).Return(int64(0), nil)
	mock.EXPECT().Migrate().Return([]MigrationError{}).Times(3)

	ctx := context.Background()

	// Act
	chunks, done := runMigration(ctx, mock)
	errs, err := readChannels(chunks, done)

	// Assert
	require.NoError(t, err)
	require.Empty(t, errs) // updated to check allChunks instead of errs
}

// Test that the migration errors are aggregated.
func TestRunMigrationAggregatesErrors(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(10))).Return(int64(5), nil)
	mock.EXPECT().LoadItems(gomock.Eq(int64(15))).Return(int64(0), nil)

	callCount := 0
	mock.EXPECT().Migrate().DoAndReturn(func() []MigrationError {
		callCount++
		return []MigrationError{
			{
				ID:    int64(callCount),
				Err:   errors.Errorf("error %d", callCount),
				Label: fmt.Sprintf("host %d", callCount),
			},
		}
	}).Times(2)

	ctx := context.Background()

	// Act
	chunks, done := runMigration(ctx, mock)
	errs, err := readChannels(chunks, done)

	// Assert
	require.NoError(t, err)
	require.Len(t, errs, 2)
	require.EqualValues(t, 1, errs[0].ID)
	require.EqualError(t, errs[0].Err, "error 1")
	require.Equal(t, "host 1", errs[0].Label)
	require.EqualValues(t, 2, errs[1].ID)
	require.EqualError(t, errs[1].Err, "error 2")
	require.Equal(t, "host 2", errs[1].Label)
}

// Test that the runner interrupts the migration after the loading error.
func TestRunMigrationInterruptOnLoadingError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockMigrator(ctrl)
	mock.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(10), errors.New("loading error"))

	ctx := context.Background()

	// Act
	chunks, done := runMigration(ctx, mock)
	errs, err := readChannels(chunks, done)

	// Assert
	require.ErrorContains(t, err, "loading error")
	require.Empty(t, errs)
}
