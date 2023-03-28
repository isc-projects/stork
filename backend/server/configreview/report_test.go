package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test creating a valid report.
func TestCreateReport(t *testing.T) {
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 123,
	}, Triggers{ConfigModified}, nil)
	referencedDaemon := &dbmodel.Daemon{
		ID: 567,
	}
	report, err := NewReport(ctx, "new report for {daemon}").
		referencingDaemon(referencedDaemon).
		referencingDaemon(ctx.subjectDaemon).
		create()
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.content)
	require.Equal(t, "new report for {daemon}", *report.content)
	require.EqualValues(t, 123, report.daemonID)
	require.Len(t, report.refDaemonIDs, 2)
	require.EqualValues(t, 567, report.refDaemonIDs[0])
	require.EqualValues(t, 123, report.refDaemonIDs[1])
}

// Test that an attempt to create a report with a blank content is
// not possible.
func TestCreateBlankReport(t *testing.T) {
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 123,
	}, Triggers{ConfigModified}, nil)
	report, err := NewReport(ctx, "   ").create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that an attempt to create a report with subject daemon ID
// of 0 is not possible.
func TestCreateZeroSubjectDaemonID(t *testing.T) {
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 0,
	}, Triggers{ConfigModified}, nil)

	report, err := NewReport(ctx, "new report").create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that an attempt to create a report with referenced daemon
// ID of 0 is not possible.
func TestCreateZeroReferencedDaemonID(t *testing.T) {
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 123,
	}, Triggers{ConfigModified}, nil)
	referencedDaemon := &dbmodel.Daemon{
		ID: 0,
	}
	report, err := NewReport(ctx, "new report").
		referencingDaemon(referencedDaemon).
		create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that referencing the same daemon twice is not possible.
func TestCreateReportRepeatedSubjectDaemon(t *testing.T) {
	daemons := []*dbmodel.Daemon{
		{
			ID: 123,
		},
		{
			ID: 234,
		},
		{
			ID: 123,
		},
	}
	ctx := newReviewContext(nil, daemons[1], Triggers{ConfigModified}, nil)

	report, err := NewReport(ctx, "new report").
		referencingDaemon(daemons[0]).
		referencingDaemon(daemons[1]).
		referencingDaemon(daemons[2]).
		create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that the empty report is constructed properly.
func TestNewEmptyReport(t *testing.T) {
	// Arrange
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 123,
	}, Triggers{ConfigModified}, nil)

	// Act
	report, err := newEmptyReport(ctx)

	// Assert
	require.NoError(t, err)
	require.Nil(t, report.content)
	require.EqualValues(t, 123, report.daemonID)
	require.Empty(t, report.refDaemonIDs)
}

// Test that an attempt to create an empty report with subject daemon ID
// of 0 is not possible.
func TestCreateEmptyReportZeroSubjectDaemonID(t *testing.T) {
	// Arrange
	ctx := newReviewContext(nil, &dbmodel.Daemon{
		ID: 0,
	}, Triggers{ConfigModified}, nil)

	// Act
	report, err := newEmptyReport(ctx)

	// Assert
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that the issue is indicated properly.
func TestReportIsIssueFound(t *testing.T) {
	// Arrange
	ctx := newReviewContext(nil, &dbmodel.Daemon{ID: 42}, Triggers{ConfigModified}, nil)
	report, _ := NewReport(ctx, "foobar").create()
	emptyReport, _ := newEmptyReport(ctx)

	// Act & Assert
	require.True(t, report.IsIssueFound())
	require.False(t, emptyReport.IsIssueFound())
}
