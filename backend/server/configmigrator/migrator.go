package configmigrator

type Migrator interface {
	CountTotal() (int64, error)
	LoadItems(offset int64) (int64, error)
	Migrate() map[int64]error
}
