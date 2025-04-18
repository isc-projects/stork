package configmigrator

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type migrationChunk struct {
	loadedCount int64
	errs        []MigrationError
	generalErr  error
}

// It is an asynchronous migration runner. It loads the items in chunks and
// migrates them until there are no more items to migrate.
// Returns an output channel with the migration progress. The channel contains
// the number of items that were migrated and the errors that occurred during
// the migration. It may contain also a general error that interrupted the
// migration. The channel is closed when the migration is finished or when an
// general error occurs.
func runMigration(ctx context.Context, migrator Migrator) <-chan migrationChunk {
	ch := make(chan migrationChunk)

	go func() {
		defer close(ch)

		// Begin the migration - make all necessary preparations.
		err := migrator.Begin()
		if err != nil {
			ch <- migrationChunk{
				generalErr: err,
			}
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
				ch <- migrationChunk{
					generalErr: ctx.Err(),
				}
				return
			default:
				loadedCount, err = migrator.LoadItems(totalLoadedCount)
				if err != nil {
					ch <- migrationChunk{
						generalErr: err,
					}
					return
				}

				if loadedCount == 0 {
					// No more items to migrate.
					return
				}

				totalLoadedCount += loadedCount

				errs := migrator.Migrate()

				ch <- migrationChunk{
					loadedCount: loadedCount, errs: errs,
				}
			}
		}
	}()

	return ch
}
