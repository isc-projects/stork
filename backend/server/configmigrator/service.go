package configmigrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Type (alias?) for the migration ID.
type MigrationIdentifier string

// Describes the migration process. It is common for all the entities that can
// be migrated.
type MigrationStatus struct {
	// Unique identifier of the migration.
	ID MigrationIdentifier
	// Name of the user who started the migration.
	StartedBy string
	// Start date and time of the migration.
	StartDate time.Time
	// End date and time of the migration. It is zero if the migration is not
	// finished yet.
	EndDate time.Time
	// Indicates if the migration is canceling.
	Canceling bool
	// Progress of the migration. The value is between 0 and 1 (finished).
	Progress float64
	// The errors already occurred during the migration. The key is the ID of
	// the migrated item which type is defined by the @EntityType field.
	Errors map[int64]error
	// The general error that interrupted the migration. This error is not
	// related to any specific item rather to the whole migration process.
	// If it is not nil, the migration is stopped but if the migration is
	// finished successfully, the value is nil.
	GeneralError error
	// The type of the entities that are being migrated.
	EntityType EntityType
	// The amount of time that has already elapsed since the migration started.
	ElapsedTime time.Duration
	// The estimated amount of time that is left to finish the migration.
	EstimatedLeftTime time.Duration
}

// Internal migration structure that holds the migration data and asynchronous
// artifacts.
// Instances of this structure are prone to race conditions because they are
// read and updated asynchronously. The structure is protected by the mutexes.
// It is recommended to use the methods of the structure to access or modify
// the data to ensure the mutexes are properly locked and unlocked.
//
// Asynchronous model of the migration.
// We have three actors in the migration process:
//  1. The external packages that want to start, stop, or get the status of the
//     migration. In the same time, multiple external callers may want to
//     interact with the migration service (e.g., RestAPI handlers).
//  2. The migration service that manages the migrations. It is responsible for
//     handling the external (outside package) requests and controlling the
//     migrations. It may receive multiple requests in parallel (e.g., from the
//     RestAPI handlers). It may also receive multiple migration updates from
//     the migration runners.
//  3. The migration runner that is responsible for a single migration. The
//     runner returns two channels on start. The first channel is used to
//     report the progress of the migration. The second channel indicates the
//     end of the migration (successful or failed).
//
// To synchronize the access to the migration data, we use the mutexes. The
// the migration service owns a single mutex that protects the storage of the
// migrations. This mutex is utilized in calls received from the external
// packages.
// Each migration owns its mutex that protects the migration data. This mutex
// is utilized in exchanging the data between the migration service and the
// migration runner.
//
//		+----------+           +-----------+             +-----------+
//		|          | Service's |           | Migration's |           |
//		| External |   mutex   | Migration |    mutex 1  | Migration |
//		| packages |<--------->| service   |<----------->| runner 1  |
//		|          |           |           |             |           |
//		+----------+           +-----------+             +-----------+
//	                                 ^
//	                                 |                   +-----------+
//	                                 |    Migration's    |           |
//	                                 |       mutex 2     | Migration |
//	                                 +------------------>| runner 2  |
//	                                                     |           |
//	                                                     +-----------+
type migration struct {
	id             MigrationIdentifier
	startedBy      string
	startDate      time.Time
	endDate        time.Time
	cancelDate     time.Time
	entityType     EntityType
	processedItems int64
	totalItems     int64
	errors         map[int64]error
	generalError   error

	// Recommended to not use this object outside of the migration structure.
	mutex sync.RWMutex
	// Function that cancels the context passed to the migration runner.
	// It doesn't interrupt the migration immediately.
	cancelFunc context.CancelFunc
}

// Returns the current status of the migration.
func (m *migration) getStatus() MigrationStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Elapsed time.
	var elapsedTime time.Duration
	if !m.endDate.IsZero() {
		elapsedTime = m.endDate.Sub(m.startDate)
	} else {
		elapsedTime = time.Since(m.startDate)
	}

	// Estimated left time.
	var estimatedLeftTime time.Duration
	if m.totalItems != 0 && m.processedItems != 0 {
		processingTimeOfSingleItem := elapsedTime / time.Duration(m.processedItems)
		leftItems := m.totalItems - m.processedItems
		estimatedLeftTime = processingTimeOfSingleItem * time.Duration(leftItems)
	}

	errsCopy := make(map[int64]error, len(m.errors))
	for k, v := range m.errors {
		errsCopy[k] = v
	}

	// It needs to return a copy of the migration data to avoid race
	// conditions.
	return MigrationStatus{
		ID:                m.id,
		StartedBy:         m.startedBy,
		StartDate:         m.startDate,
		EndDate:           m.endDate,
		EntityType:        m.entityType,
		Canceling:         !m.cancelDate.IsZero(),
		Errors:            errsCopy,
		GeneralError:      m.generalError,
		Progress:          float64(m.processedItems) / float64(m.totalItems),
		ElapsedTime:       elapsedTime,
		EstimatedLeftTime: estimatedLeftTime,
	}
}

// Registers that the stopping of the migration is done. The reasonErr is the
// error that caused the migration to stop. If the migration is finished
// successfully, the reasonErr is nil.
func (m *migration) registerStop(reasonErr error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.endDate = time.Now()
	m.generalError = reasonErr
}

// Requests the migration to stop. The migration is stopped asynchronously.
func (m *migration) cancel() {
	m.mutex.RLock()
	if !m.cancelDate.IsZero() || !m.endDate.IsZero() {
		// The migration is already canceled or finished.
		m.mutex.RUnlock()
		return
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	m.cancelDate = time.Now()
	m.mutex.Unlock()

	m.cancelFunc()
}

// Registers the chunk of the loaded items. The loadedItems is the number of
// items that were loaded in the chunk. The errs is a map of errors that
// occurred during the migration of the items. The key is the ID of the item.
func (m *migration) registerChunk(loadedItems int64, errs map[int64]error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.processedItems += loadedItems
	for id, err := range errs {
		m.errors[id] = err
	}
}

// Migration service interface. It provides the methods to interact with the
// migrations. The service is responsible for starting and stopping the
// migrations. It also provides the information about the migrations.
type Service interface {
	// Returns the list of all migrations.
	GetMigrations() []MigrationStatus
	// Returns the migration with the provided ID. If the migration is not found,
	// the second return value is false.
	GetMigration(id MigrationIdentifier) (MigrationStatus, bool)
	// Starts the migration in background. The migrator is the object that knows how to migrate
	// certain entities. The username is the name of the user who started the
	// migration.
	StartMigration(migrator Migrator, username string) (MigrationStatus, error)
	// Requests the migration to stop. The migration is stopped asynchronously.
	// If the migration is not found, the second return value is false.
	StopMigration(id MigrationIdentifier) (MigrationStatus, bool)
	// Clears the finished migrations from the memory.
	ClearFinishedMigrations()
}

// It manages the migrations. It is responsible for starting and stopping the
// migrations. It also provides the information about the migrations.
//
// The migration data are stored in memory only. The data are lost when the
// server is restarted.
type service struct {
	migrations map[MigrationIdentifier]*migration
	mutex      sync.RWMutex
}

// Constructs a new migration service.
func NewService() Service {
	return &service{
		migrations: make(map[MigrationIdentifier]*migration),
	}
}

// Returns the list of all migrations.
func (s *service) GetMigrations() []MigrationStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var statuses []MigrationStatus
	for _, m := range s.migrations {
		statuses = append(statuses, m.getStatus())
	}
	return statuses
}

// Returns the migration with the provided ID. If the migration is not found,
// the second return value is false.
func (s *service) GetMigration(id MigrationIdentifier) (MigrationStatus, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	migration, ok := s.migrations[id]
	if !ok {
		return MigrationStatus{}, false
	}
	return migration.getStatus(), true
}

// Starts the migration in background. The migrator is the object that knows how to migrate
// certain entities. The username is the name of the user who started the
// migration.
func (s *service) StartMigration(migrator Migrator, username string) (MigrationStatus, error) {
	totalItems, err := migrator.CountTotal()
	if err != nil {
		return MigrationStatus{}, errors.WithMessage(err, "failed to get the total items")
	}

	ctx, cancel := context.WithCancel(context.Background())

	migration := &migration{
		startedBy:      username,
		startDate:      time.Now(),
		entityType:     migrator.GetEntityType(),
		processedItems: 0,
		totalItems:     totalItems,
		errors:         make(map[int64]error),
		cancelFunc:     cancel,
	}

	// Run migration.
	chunkChunk, doneChan := runMigration(ctx, migrator)

	go func() {
		for {
			select {
			case migratedCount := <-chunkChunk:
				migration.registerChunk(migratedCount.loadedCount, migratedCount.errs)
			case err := <-doneChan:
				migration.registerStop(err)
			}
		}
	}()

	// Save the migration.
	s.mutex.Lock()
	migration.id = s.getUniqueMigrationID(migration.startDate)
	s.migrations[migration.id] = migration
	s.mutex.Unlock()

	return migration.getStatus(), nil
}

// Generates a unique migration ID based on the provided date.
func (s *service) getUniqueMigrationID(date time.Time) MigrationIdentifier {
	iteration := 1
	for {
		id := MigrationIdentifier(fmt.Sprintf("%d-%d", date.Unix(), iteration))
		if _, ok := s.migrations[id]; !ok {
			return id
		}
		iteration++
	}
}

// Requests the migration to stop. The migration is stopped asynchronously.
// If the migration is not found, the second return value is false.
func (s *service) StopMigration(id MigrationIdentifier) (MigrationStatus, bool) {
	s.mutex.RLock()
	migration, ok := s.migrations[id]
	s.mutex.RUnlock()

	if !ok {
		return MigrationStatus{}, false
	}

	migration.cancel()
	return migration.getStatus(), true
}

// Clears the finished migrations from the memory.
func (s *service) ClearFinishedMigrations() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, m := range s.migrations {
		if !m.endDate.IsZero() {
			delete(s.migrations, m.id)
		}
	}
}
