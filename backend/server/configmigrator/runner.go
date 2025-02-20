package configmigrator

import (
	"context"
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

		var totalLoadedCount int64
		var loadedCount int64
		var err error
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
