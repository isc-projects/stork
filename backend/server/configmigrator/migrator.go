package configmigrator

// Indicates the entities that are being migrated.
type EntityType string

const (
	EntityTypeHost EntityType = "host"
)

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
