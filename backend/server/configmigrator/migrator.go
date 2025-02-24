package configmigrator

// Indicates the entity related to the migration error. Usually it is an entity
// that was migrated or its daemon.
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
	// Returns a total number of items to migrate.
	CountTotal() (int64, error)
	// Loads a chunk of items from the Kea configuration. Returns the number of
	// loaded items and an error if any.
	LoadItems(offset int64) (int64, error)
	// Migrates the loaded items. Returns a map of errors that occurred during
	// the migration. The key is the ID of the migrated item.
	Migrate() []MigrationError
	// Indicates the type of the entities that are being migrated. The keys of
	// the error map returned by the @Migrate method are the IDs of the
	// entities of this type.
	GetEntityType() EntityType
}
