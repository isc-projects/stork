package configmigrator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

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
	}
	// General error occurred.
	service.migrations["general-error"] = &migration{
		id:             "general-error",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 5, 11, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   errors.New("general error"),
		processedItems: 10,
		totalItems:     100,
		errors:         make(map[int64]error),
	}

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
