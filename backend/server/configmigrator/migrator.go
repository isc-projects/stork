package configmigrator

// Indicates a type of entity for which migration error is reported. Usually, it is
// a migrated entity or a corresponding daemon.
type EntityType string

const (
	EntityTypeHost   EntityType = "host"
	EntityTypeDaemon EntityType = "daemon"
)

// Contains the basic information about the entity type that was failed to
// migrate.
type MigrationError struct {
	// ID of the errored entity.
	ID int64
	// Label of the errored entity.
	Label string
	// Type of the errored entity.
	Type EntityType
	// Error that occurred during the migration.
	Error error
}

// Interface implemented by the structs that know how to migrate the particular
// entries from the Kea configuration to the database.
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
	LoadItems(offset int64) (int64, error)
	// Migrates the loaded items. Returns a map of errors that occurred during
	// the migration. The key is the ID of the migrated item.
	Migrate() []MigrationError
}
