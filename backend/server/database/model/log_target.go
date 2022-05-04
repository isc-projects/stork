package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
)

// A structure reflecting information about a logger used by a daemon.
type LogTarget struct {
	ID        int64 // Logger ID
	Name      string
	Severity  string
	Output    string
	CreatedAt time.Time

	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
}

// Retrieves log target from the database by id.
func GetLogTargetByID(db *pg.DB, id int64) (*LogTarget, error) {
	logTarget := LogTarget{}
	err := db.Model(&logTarget).
		Relation("Daemon.App.Machine").
		Where("log_target.id = ?", id).
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting log target with ID %d", id)
	}
	return &logTarget, nil
}
