package dumps

import (
	"github.com/go-pg/pg/v9"
	dbmodel "isc.org/stork/server/database/model"
)

// The dump of the global Stork Server settings.
type SettingsDump struct {
	BasicDump
	db *pg.DB
}

func NewSettingsDump(db *pg.DB) *SettingsDump {
	return &SettingsDump{
		*NewBasicDump("server-settings"),
		db,
	}
}

func (d *SettingsDump) Execute() error {
	settings, err := dbmodel.GetAllSettings(d.db)
	if err != nil {
		return err
	}

	d.artifacts = append(d.artifacts, NewBasicStructArtifact(
		"all", settings,
	))
	return nil
}
