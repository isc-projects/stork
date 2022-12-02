package dbmodel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Registers M:N SQL relations defined in this file.
func init() {
	orm.RegisterTable((*DaemonToConfigReport)(nil))
}

// Structure representing a single config report generated during
// the daemons configuration review.
type ConfigReport struct {
	ID          int64
	CreatedAt   time.Time
	CheckerName string
	Content     *string `pg:",use_zero"`

	DaemonID int64

	RefDaemons []*Daemon `pg:"many2many:daemon_to_config_report,fk:config_report_id,join_fk:daemon_id"`
}

// Returns true if an issue was found.
func (r *ConfigReport) IsIssueFound() bool {
	return r.Content != nil
}

// Structure representing a many-to-many relationship between daemons
// and config reports.
type DaemonToConfigReport struct {
	DaemonID       int64 `pg:",pk"`
	ConfigReportID int64 `pg:",pk"`
	OrderIndex     int64
}

// Adds a single configuration report and its relationships with the
// daemons to the database in a transaction.
func addConfigReport(tx *pg.Tx, configReport *ConfigReport) error {
	if configReport.IsIssueFound() && *configReport.Content == "" {
		return pkgerrors.Errorf("config review content cannot be empty")
	}

	// Insert the config_report entry.
	_, err := tx.Model(configReport).Insert()

	if err == nil {
		// Insert associations between the configuration report and
		// the daemons.
		var assocs []DaemonToConfigReport
		for i := range configReport.RefDaemons {
			d := configReport.RefDaemons[i]
			assocs = append(assocs, DaemonToConfigReport{
				DaemonID:       d.ID,
				ConfigReportID: configReport.ID,
				OrderIndex:     int64(i),
			})
		}

		if len(assocs) > 0 {
			// Insert the associations.
			_, err = tx.Model(&assocs).OnConflict("DO NOTHING").Insert()
		}
	}

	if err != nil {
		// The error message is formatted differently depending on whether we
		// have one or more daemons associated with the config report.
		if len(configReport.RefDaemons) == 1 {
			err = pkgerrors.Wrapf(err, "problem inserting the configuration report for daemon %d",
				configReport.RefDaemons[0].ID)
		} else {
			var daemonIds []string
			for _, d := range configReport.RefDaemons {
				daemonIds = append(daemonIds, fmt.Sprintf("%d", d.ID))
			}
			err = pkgerrors.Wrapf(err, "problem inserting the configuration report for daemons %s",
				strings.Join(daemonIds, ", "))
		}
		return err
	}

	return nil
}

// Adds a single configuration report and its relationships with the
// daemons to the database.
func AddConfigReport(dbi dbops.DBI, configReport *ConfigReport) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addConfigReport(tx, configReport)
		})
	}
	return addConfigReport(dbi.(*pg.Tx), configReport)
}

// Select all or a range of the config reports for the specified daemon.
// The offset of 0 causes the function to return reports beginning from
// the first one for the daemon. The limit of 0 causes the function to
// return all reports beginning from the offset. A non-zero limit value
// limits the number of returned reports. Specify an offset and limit of
// 0 to fetch all reports for a daemon. Besides returning the config
// reports this function also returns the total number of reports for
// the daemon (useful when paging the results) and an error.
// If the issuesOnly flag is true, it returns the reports containing
// actual issues. The reports that detected no issues are not returned.
func GetConfigReportsByDaemonID(db *pg.DB, offset, limit int64, daemonID int64, issuesOnly bool) ([]ConfigReport, int64, error) {
	var configReports []ConfigReport
	q := db.Model(&configReports).
		Where("config_report.daemon_id = ?", daemonID)

	if issuesOnly {
		q = q.Where("config_report.content IS NOT NULL")
	}

	q = q.Order("config_report.id ASC").
		Relation("RefDaemons", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("daemon_to_config_report.order_index ASC"), nil
		}).
		Relation("RefDaemons.App").
		Offset(int(offset))

	if limit != 0 {
		q = q.Limit(int(limit))
	}

	total, err := q.SelectAndCount()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem selecting config reports for daemon %d", daemonID)
		return configReports, 0, err
	}
	return configReports, int64(total), nil
}

// Counts the total number of config reports. Accepts the same filters as
// GetConfigReportsByDaemonID.
func CountConfigReportsByDaemonID(db *pg.DB, daemonID int64, issuesOnly bool) (int64, error) {
	q := db.Model((*ConfigReport)(nil)).
		Where("config_report.daemon_id = ?", daemonID)

	if issuesOnly {
		q = q.Where("config_report.content IS NOT NULL")
	}

	total, err := q.Count()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem counting config reports for daemon %d", daemonID)
		return 0, err
	}

	return int64(total), nil
}

// Delete all config reports for the specified daemon.
func DeleteConfigReportsByDaemonID(dbi dbops.DBI, daemonID int64) error {
	_, err := dbi.Model((*ConfigReport)(nil)).
		Where("daemon_id = ?", daemonID).
		Delete()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting config reports for daemon %d", daemonID)
	}

	return err
}

// A go-pg hook executed after selecting the config reports. It fills the
// daemon placeholders with the tags that can be later turned into the links
// to the daemons.
func (r *ConfigReport) AfterSelect(ctx context.Context) error {
	if !r.IsIssueFound() {
		// A report about finding no issues has no content.
		return nil
	}

	for _, daemon := range r.RefDaemons {
		content := strings.Replace(*r.Content, "{daemon}",
			fmt.Sprintf("<daemon id=\"%d\" name=\"%s\" appId=\"%d\" appType=\"%s\">",
				daemon.ID, daemon.Name, daemon.AppID, daemon.App.Type),
			1)
		r.Content = &content
	}
	return nil
}
