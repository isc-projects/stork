package dump

import (
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// The dump of the global Stork Server settings.
type SettingsDump struct {
	BasicDump
	db *pg.DB
}

// Construct the server settings dump.
func NewSettingsDump(db *pg.DB) *SettingsDump {
	return &SettingsDump{
		*NewBasicDump("server-settings"),
		db,
	}
}

// It just dumps the setting DB table content.
func (d *SettingsDump) Execute() error {
	settings, err := dbmodel.GetAllSettings(d.db)
	if err != nil {
		return err
	}

	d.AppendArtifact(NewBasicStructArtifact(
		"all", settings,
	))
	return nil
}
