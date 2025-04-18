package configmigrator

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Type (alias?) for the migration ID.
type MigrationIdentifier int64

// Describes the migration process. It is common for all the entities that can
// be migrated.
type MigrationStatus struct {
	// Unique identifier of the migration.
	ID MigrationIdentifier
	// The context passed when the migration was started.
	Context context.Context
	// Start date and time of the migration.
	StartDate time.Time
	// End date and time of the migration. It is zero if the migration is not
	// finished yet.
	EndDate time.Time
	// Indicates if the migration is canceling.
	Canceling bool
	// Number of already processed items.
	ProcessedItemsCount int64
	// Total number of items that should be migrated.
	TotalItemsCount int64
	// The errors already occurred during the migration. The key is the ID of
	// the migrated item which type is defined by the @EntityType field.
	Errors []MigrationError
	// The general error that interrupted the migration. This error is not
	// related to any specific item rather to the whole migration process.
	// If it is not nil, the migration is stopped but if the migration is
	// finished successfully, the value is nil.
	GeneralError error
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
//     interact with the migration manager (e.g., RestAPI handlers).
//  2. The migration manager that manages the migrations. It is responsible for
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
// the migration manager owns a single mutex that protects the storage of the
// migrations. This mutex is utilized in calls received from the external
// packages.
// Each migration owns its mutex that protects the migration data. This mutex
// is utilized in exchanging the data between the migration manager and the
// migration runner.
//
//		+----------+           +-----------+             +-----------+
//		|          | Manager's |           | Migration's |           |
//		| External |   mutex   | Migration |    mutex 1  | Migration |
//		| packages |<--------->| manager   |<----------->| runner 1  |
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
	ctx            context.Context
	startDate      time.Time
	endDate        time.Time
	cancelDate     time.Time
	processedItems int64
	totalItems     int64
	errors         []MigrationError
	generalError   error

	// Recommended to not use this object outside of the migration structure.
	mutex sync.RWMutex
	// Function that cancels the context passed to the migration runner.
	// It doesn't interrupt the migration immediately.
	cancelFunc context.CancelFunc
	// Emits a value when the migration is done (successful or failed). It is
	// guaranteed that when the channel is closed, the migration goroutine
	// is gone.
	doneChan <-chan struct{}
}

// Returns the current status of the migration.
func (m *migration) getStatus() *MigrationStatus {
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

	errsCopy := make([]MigrationError, len(m.errors))
	copy(errsCopy, m.errors)

	// It needs to return a copy of the migration data to avoid race
	// conditions.
	return &MigrationStatus{
		ID: m.id,
		// It should be safe to return the context with cancellation or
		// deadline but I'm not sure if it is a good idea. Let's return the
		// context with just the data provided to the start migration method.
		Context:             context.WithoutCancel(m.ctx),
		StartDate:           m.startDate,
		EndDate:             m.endDate,
		Canceling:           !m.cancelDate.IsZero(),
		Errors:              errsCopy,
		GeneralError:        m.generalError,
		ProcessedItemsCount: m.processedItems,
		TotalItemsCount:     m.totalItems,
		ElapsedTime:         elapsedTime,
		EstimatedLeftTime:   estimatedLeftTime,
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
// items that were loaded in the chunk. The errs is a slice of errors that
// occurred during the migration of the items.
func (m *migration) registerChunk(loadedItems int64, errs []MigrationError) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.processedItems += loadedItems
	m.errors = append(m.errors, errs...)
}

// Migration manager interface. It provides the methods to interact with the
// migrations. The manager is responsible for starting and stopping the
// migrations. It also provides the information about the migrations.
type MigrationManager interface {
	// Returns the list of all migrations.
	GetMigrations() []*MigrationStatus
	// Returns the migration with the provided ID. If the migration is not found,
	// the second return value is false.
	GetMigration(id MigrationIdentifier) (*MigrationStatus, bool)
	// Starts the migration in background. The migrator is the object that knows how to migrate
	// certain entities. The context may contain the information about the user
	// who started the migration.
	StartMigration(ctx context.Context, migrator Migrator) (*MigrationStatus, error)
	// Requests the migration to stop. The migration is stopped asynchronously.
	// If the migration is not found, the second return value is false.
	StopMigration(id MigrationIdentifier) (*MigrationStatus, bool)
	// Clears the finished migrations from the memory.
	ClearFinishedMigrations()
	// Cancels all the running migrations and waits for them to finish.
	// The method is blocking.
	Close()
}

// It manages the migrations. It is responsible for starting and stopping the
// migrations. It also provides the information about the migrations.
//
// The migration data are stored in memory only. The data are lost when the
// server is restarted.
type manager struct {
	migrations      map[MigrationIdentifier]*migration
	mutex           sync.RWMutex
	nextMigrationID MigrationIdentifier
}

// Constructs a new migration manager.
func NewMigrationManager() MigrationManager {
	return &manager{
		migrations:      make(map[MigrationIdentifier]*migration),
		nextMigrationID: 1,
	}
}

// Returns the list of all migrations sorted by the start date.
func (s *manager) GetMigrations() []*MigrationStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var statuses []*MigrationStatus
	for _, m := range s.migrations {
		statuses = append(statuses, m.getStatus())
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].StartDate.Before(statuses[j].StartDate)
	})

	return statuses
}

// Returns the migration with the provided ID. If the migration is not found,
// the second return value is false.
func (s *manager) GetMigration(id MigrationIdentifier) (*MigrationStatus, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	migration, ok := s.migrations[id]
	if !ok {
		return nil, false
	}
	return migration.getStatus(), true
}

// Starts the migration in background. The migrator is the object that knows how to migrate
// certain entities. The context may contain the information about the user
// who started the migration.
func (s *manager) StartMigration(ctx context.Context, migrator Migrator) (*MigrationStatus, error) {
	totalItems, err := migrator.CountTotal()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get the total items")
	}

	// Strip the cancel and deadline from the parent context.
	ctx = context.WithoutCancel(ctx)
	// Add own independent cancel.
	ctx, cancel := context.WithCancel(ctx)

	// Emits a value when the runner is done and its done value has been
	// processed.
	workerDoneChan := make(chan struct{})

	migration := &migration{
		ctx:            ctx,
		startDate:      time.Now(),
		processedItems: 0,
		totalItems:     totalItems,
		errors:         make([]MigrationError, 0),
		cancelFunc:     cancel,
		doneChan:       workerDoneChan,
	}

	// Save the migration.
	s.mutex.Lock()
	migrationID := s.generateUniqueMigrationID()
	migration.id = migrationID
	s.migrations[migration.id] = migration
	s.mutex.Unlock()

	// Run migration.
	log.WithFields(log.Fields{
		"migrationID": migrationID,
		"totalItems":  totalItems,
	}).Info("Starting config migration")
	chunkChan := runMigration(ctx, migrator)

	go func() {
		defer close(workerDoneChan)
		for chunk := range chunkChan {
			if chunk.generalErr != nil {
				migration.registerStop(chunk.generalErr)
				return
			}
			migration.registerChunk(chunk.loadedCount, chunk.errs)
		}
		log.WithField("migrationID", migrationID).Info("Config migration done")
		migration.registerStop(nil)
	}()

	return migration.getStatus(), nil
}

// Generates a unique migration ID.
func (s *manager) generateUniqueMigrationID() MigrationIdentifier {
	id := s.nextMigrationID
	s.nextMigrationID++
	return id
}

// Requests the migration to stop. The migration is stopped asynchronously.
// If the migration is not found, the second return value is false.
func (s *manager) StopMigration(id MigrationIdentifier) (*MigrationStatus, bool) {
	s.mutex.RLock()
	migration, ok := s.migrations[id]
	s.mutex.RUnlock()

	if !ok {
		return nil, false
	}

	migration.cancel()
	return migration.getStatus(), true
}

// Clears the finished migrations from the memory.
func (s *manager) ClearFinishedMigrations() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, m := range s.migrations {
		if !m.endDate.IsZero() {
			delete(s.migrations, m.id)
		}
	}
}

// Cancels all the running migrations and waits for them to finish.
// The method is blocking.
func (s *manager) Close() {
	for _, migration := range s.migrations {
		migration.cancel()
	}

	// Wait for the migrations to finish.
	for _, migration := range s.migrations {
		<-migration.doneChan
	}
}
