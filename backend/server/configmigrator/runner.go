package configmigrator

import (
	"github.com/pkg/errors"
)

// It is a simple migration runner that is backed by the usual loop and do
// everything in foreground.
// It loads the items in chunks and migrates them until there are no more
// items to migrate.
// Returns a map of errors that occurred during the migration. The key is the
// ID of the migrated item.
func RunMigration(migrator Migrator) (map[int64]error, error) {
	migrationErrs := make(map[int64]error)
	totalCount := int64(0)

	for {
		loadedCount, err := migrator.LoadItems(totalCount)
		if err != nil {
			err = errors.WithMessage(err, "failed to load items to migrate")
			return migrationErrs, err
		}

		if loadedCount == 0 {
			// No more items to migrate.
			break
		}

		errs := migrator.Migrate()
		for id, err := range errs {
			migrationErrs[id] = err
		}

		totalCount += loadedCount
	}

	return migrationErrs, nil
}
