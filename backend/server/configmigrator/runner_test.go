package configmigrator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -package=configmigrator -destination=migratormock_test.go isc.org/stork/server/configmigrator Migrator

func readChannels(ch <-chan migrationChunk, done <-chan error) (errs map[int64]error, err error) {
	errs = make(map[int64]error)

	for {
		select {
		case err = <-done:
			return
		case chunk := <-ch:
			for id, e := range chunk.errs {
				errs[id] = e
			}
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
	mock.EXPECT().Migrate().Return(map[int64]error{}).Times(3)

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
	mock.EXPECT().Migrate().DoAndReturn(func() map[int64]error {
		callCount++
		return map[int64]error{
			int64(callCount): errors.Errorf("error %d", callCount),
		}
	}).Times(2)

	ctx := context.Background()

	// Act
	chunks, done := runMigration(ctx, mock)
	errs, err := readChannels(chunks, done)

	// Assert
	require.NoError(t, err)
	require.Len(t, errs, 2)
	require.EqualError(t, errs[1], "error 1")
	require.EqualError(t, errs[2], "error 2")
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
