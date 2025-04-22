package configmigrator

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type migrationChunkStatus struct {
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
func runMigration(ctx context.Context, migrator Migrator) <-chan migrationChunkStatus {
	ch := make(chan migrationChunkStatus)

	go func() {
		// Begin the migration - make all necessary preparations.
		err := migrator.Begin()
		if err != nil {
			ch <- migrationChunkStatus{
				generalErr: err,
			}
			close(ch)
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
			close(ch)
		}()

		for {
			select {
			case <-ctx.Done():
				ch <- migrationChunkStatus{
					generalErr: ctx.Err(),
				}
				return
			default:
				loadedCount, err := migrator.LoadItems()
				if err != nil {
					ch <- migrationChunkStatus{
						generalErr: err,
					}
					return
				}

				if loadedCount == 0 {
					// No more items to migrate.
					return
				}

				errs := migrator.Migrate()

				ch <- migrationChunkStatus{
					loadedCount: loadedCount, errs: errs,
				}
			}
		}
	}()

	return ch
}
