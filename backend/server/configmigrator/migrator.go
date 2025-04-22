package configmigrator

// Indicates a type of entity for which migration error is reported. Usually, it is
// a migrated entity or a corresponding daemon.
type ErrorCauseEntity string

const (
	// Indicates that a specific host entity failed to migrate.
	ErrorCauseEntityHost ErrorCauseEntity = "host"
	// The daemons are not migrated itself, but the daemon is
	// responsible for processing commands that migrate an entity. The
	// communication with the daemon can fail in various ways. It causes the
	// migration of the entity managed by the daemon to be blocked. In such
	// situations, this type is used to report the error.
	// Alternatively, we could create an error for each entity that is managed
	// by the daemon, but it would be less efficient because it would produce
	// a lot of errors with the same error message as all of them would have
	// the same root cause.
	ErrorCauseEntityDaemon ErrorCauseEntity = "daemon"
)

// Contains the basic information about the entity type that was failed to
// migrate.
type MigrationError struct {
	// ID of the errored entity.
	ID int64
	// Label of the errored entity.
	Label string
	// Type of the errored entity.
	CauseEntity ErrorCauseEntity
	// Error that occurred during the migration.
	Error error
}

// Interface implemented by the structs migrating entries (typically
// configuration elements) between different storages. One of the examples is
// the migration of the Kea configuration elements from Kea configuration file
// to a database backend.
type Migrator interface {
	// Begins the migration. Returns an error if the migration cannot be
	// started. It is called before the first LoadItems call.
	Begin() error
	// Ends the migration. It is called after the last Migrate call.
	End() error
	// Returns a total number of items to migrate.
	CountTotal() (int64, error)
	// Loads a chunk of items from the Kea configuration. Returns the number of
	// loaded items and an error if any.
	LoadItems() (int64, error)
	// Migrates the loaded items. Returns a map of errors that occurred during
	// the migration. The key is the ID of the migrated item.
	Migrate() []MigrationError
}
