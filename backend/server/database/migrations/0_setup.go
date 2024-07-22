package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

// This init function is called first in this package because the filename
// is first in the lexical order. It should be used to configure the migration
// framework.
func init() {
	// Disable autodiscover for SQL migration files. It prevents the migration
	// framework from looking for Stork source files in production
	// environments. This is necessary because the look-up process can fail
	// due to insufficient permissions to read particular directories.
	migrations.DefaultCollection.DisableSQLAutodiscover(true)
}
