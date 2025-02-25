package configmigrator

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type migrationChunk struct {
	loadedCount int64
	errs        []MigrationError
}

// It is an asynchronous migration runner. It loads the items in chunks and
// migrates them until there are no more items to migrate.
// Returns the channel with the total number of migrated items that are sent
// after each chunk of items is migrated, the channel with the errors that
// occurred during the migration, and the done channel that is closed when the
// migration is finished, it may contain an error if the migration was
// interrupted by a general error.
func runMigration(ctx context.Context, migrator Migrator) (<-chan migrationChunk, <-chan error) {
	migrationChunkChan := make(chan migrationChunk)
	doneChan := make(chan error)

	go func() {
		defer close(migrationChunkChan)
		defer close(doneChan)

		// Begin the migration - make all necessary preparations.
		err := migrator.Begin()
		if err != nil {
			doneChan <- err
			return
		}
		defer func() {
			// End the migration. Clean up after the migration.
			err := migrator.End()
			if err != nil {
				// We cannot return this error through the done channel because
				// the done channel is expected to return exactly one value.
				// We log the error instead as it is not a problem with the
				// migrated data but with the migration runner itself.
				log.WithError(err).Error("failed to clean up after migration")
			}
		}()

		var totalLoadedCount int64
		var loadedCount int64
		for {
			select {
			case <-ctx.Done():
				doneChan <- ctx.Err()
				return
			default:
				loadedCount, err = migrator.LoadItems(totalLoadedCount)
				if err != nil {
					doneChan <- err
					return
				}
				totalLoadedCount += loadedCount

				if loadedCount == 0 {
					// No more items to migrate.
					doneChan <- nil // indicate success if no more items to migrate
					return
				}

				errs := migrator.Migrate()

				migrationChunkChan <- migrationChunk{
					loadedCount: loadedCount, errs: errs,
				}
			}
		}
	}()

	return migrationChunkChan, doneChan
}
