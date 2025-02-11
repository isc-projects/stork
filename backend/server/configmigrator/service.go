package configmigrator

import (
	"time"

	"github.com/pkg/errors"
)

// Type (alias?) for the migration ID.
type MigrationIdentifier int64

// Indicates the entities that are being migrated.
type EntityType string

const (
	EntityTypeHost EntityType = "host"
)

// Describes the migration process. It is common for all the entities that can
// be migrated.
type Migration struct {
	// Unique identifier of the migration.
	ID MigrationIdentifier
	// Name of the user who started the migration.
	StartedBy string
	// Start date and time of the migration.
	StartDate time.Time
	// End date and time of the migration. It is zero if the migration is not
	// finished yet.
	EndDate time.Time
	// Progress of the migration. The value is between 0 and 1 (finished).
	Progress float64
	// The errors already occurred during the migration. The key is the ID of
	// the migrated item which type is defined by the @EntityType field.
	Errors map[int64]error
	// The type of the entities that are being migrated.
	EntityType EntityType
	// The amount of time that has already elapsed since the migration started.
	ElapsedTime time.Duration
	// The estimated amount of time that is left to finish the migration.
	EstimatedLeftTime time.Duration
}

// It manages the migrations. It is responsible for starting and stopping the
// migrations. It also provides the information about the migrations.
//
// The migration data are stored in memory only. The data are lost when the
// server is restarted.
type Service struct {
}

func (s *Service) GetMigrations() []Migration {
	return nil
}

func (s *Service) GetMigration(id MigrationIdentifier) (Migration, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) StartMigration(migrator Migrator) (Migration, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) StopMigration(id MigrationIdentifier) error {
	return errors.New("not implemented")
}

func (s *Service) ClearFinishedMigrations() error {
	return errors.New("not implemented")
}
