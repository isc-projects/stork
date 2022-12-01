package configreview

import (
	"strings"

	pkgerrors "github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Represents a single config review report. It may contain a description
// of one issue found during a configuration review. The daemonID field
// comprises an ID of the daemon for which the review is conducted.
// The refDaemonIDs slice contain IDs of the daemons referenced in the
// review. Each daemon can be referenced at most once. The presence of
// the referenced daemons may trigger cascaded/internal reviews. See
// the dispatcher documentation.
type Report struct {
	content      *string
	daemonID     int64
	refDaemonIDs []int64
}

// Indicates that the report contains a found issue.
func (r *Report) IsIssueFound() bool {
	return r.content != nil
}

// Represents an intermediate report which hasn't been validated yet.
type IntermediateReport Report

// Create new report. The report is associated with the subject daemon
// (a daemon for which the review is conducted) and includes an issue
// description. Additional functions can be called for this instance to
// add supplementary information to this report. For example, calling
// the referencingDaemon() function associates the report with the
// specified daemon. The report must not be used until create() function
// is called which sanity checks the report contents. An example usage:
//
//	report, err := newReport(ctx, "some issue for {daemon} and {daemon}").
//					referencingDaemon(daemon1).
//					referencingDaemon(daemon2).
//					create()
//
// When the report is later fetched from the database it is possible to
// use the referenced daemons to replace the {daemon} placeholders with
// the detailed daemon information. See the similar mechanism implemented
// in the eventcenter.
func NewReport(ctx *ReviewContext, content string) *IntermediateReport {
	content = strings.TrimSpace(content)
	return &IntermediateReport{
		content:  &content,
		daemonID: ctx.subjectDaemon.ID,
	}
}

// Creates a new empty report. This report has nil content that indicates a
// given checker found no issues.
// The checkers don't create this report directly. The empty report is
// created internally by the dispatcher. The checkers return reports
// only when they find issues.
func newEmptyReport(ctx *ReviewContext) (*Report, error) {
	// Ensure that the subject daemon has non-zero ID.
	if ctx.subjectDaemon.ID == 0 {
		return nil, pkgerrors.New("ID of the daemon for which a config report is created must not be 0")
	}

	return &Report{
		content:      nil,
		daemonID:     ctx.subjectDaemon.ID,
		refDaemonIDs: []int64{},
	}, nil
}

// Associates a report with a daemon. Do not associate the same daemon
// with the report multiple times. It will result in an error while
// calling create().
func (r *IntermediateReport) referencingDaemon(daemon *dbmodel.Daemon) *IntermediateReport {
	r.refDaemonIDs = append(r.refDaemonIDs, daemon.ID)
	return r
}

// Validates the report contents and return an instance of the final
// report or an error. It should never report an error if the checkers
// generating the reports are implemented properly.
func (r *IntermediateReport) create() (*Report, error) {
	// Ensure that the content is not blank.
	if r.content == nil || len(*r.content) == 0 {
		return nil, pkgerrors.New("config review report must not be blank")
	}

	// Ensure that the subject daemon has non-zero ID.
	if r.daemonID == 0 {
		return nil, pkgerrors.New("ID of the daemon for which a config report is created must not be 0")
	}

	// Ensure that each daemon is referenced at most once and it has
	// non-zero ID.
	presentDaemons := make(map[int64]bool)
	for _, id := range r.refDaemonIDs {
		if id == 0 {
			return nil, pkgerrors.New("config review report must not reference a daemon with ID of 0")
		}
		if _, exists := presentDaemons[id]; exists {
			return nil, pkgerrors.Errorf("config review report must not reference the same daemon %d twice", id)
		}
		presentDaemons[id] = true
	}
	// Everything is fine.
	rc := &Report{
		content:      r.content,
		daemonID:     r.daemonID,
		refDaemonIDs: r.refDaemonIDs,
	}
	return rc, nil
}
