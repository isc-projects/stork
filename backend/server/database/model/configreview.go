package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Holds the data related to the most recent configuration review pass
// for a daemon. It has one-to-one relationship with the daemon table.
// The data held in this table comprise the last configuration review
// timestamp, hash string of the configuration for which the review
// was conducted and the review dispatcher signature. The hash and
// the signature are used to determine whether the review results are
// up-to-date or a new review should be conducted (due to configuration
// changes following the last review or due to the recent updates to
// the dispatcher logic, e.g., when new checkers were implemented).
type ConfigReview struct {
	ID         int64
	CreatedAt  time.Time
	ConfigHash string
	Signature  string

	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
}

// Upserts the configuration review entry for a daemon.
func AddConfigReview(dbIface interface{}, configReview *ConfigReview) error {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	// Insert the config_review entry. If the entry exists for the daemon,
	// replace it with a new entry.
	_, err = tx.Model(configReview).
		OnConflict("(daemon_id) DO UPDATE").
		Set("created_at = EXCLUDED.created_at").
		Set("config_hash = EXCLUDED.config_hash").
		Set("signature = EXCLUDED.signature").
		Insert()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem with upserting the configuration review entry for daemon %d",
			configReview.DaemonID)
	}
	err = commit()
	return err
}

// Fetches configuration review information by daemon id.
func GetConfigReviewByDaemonID(db *pg.DB, daemonID int64) (*ConfigReview, error) {
	configReview := &ConfigReview{}
	err := db.Model(configReview).
		Relation("Daemon.KeaDaemon").
		Where("config_review.daemon_id = ?", daemonID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			// The review entry doesn't exist for the daemon, which is fine.
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with selecting the config review for daemon %d", daemonID)
		return nil, err
	}
	return configReview, nil
}
