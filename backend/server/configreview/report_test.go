package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test creating a valid report.
func TestCreateReport(t *testing.T) {
	ctx := newReviewContext()
	ctx.subjectDaemon = &dbmodel.Daemon{
		ID: 123,
	}
	referencedDaemon := &dbmodel.Daemon{
		ID: 567,
	}
	report, err := newReport(ctx, "new report for {daemon}").
		referencingDaemon(referencedDaemon).
		referencingDaemon(ctx.subjectDaemon).
		create()
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, "new report for {daemon}", report.issue)
	require.EqualValues(t, 123, report.daemon)
	require.Len(t, report.refDaemons, 2)
	require.EqualValues(t, 567, report.refDaemons[0])
	require.EqualValues(t, 123, report.refDaemons[1])
}

// Test that an attempt to create a report with a blank issue is
// not possible.
func TestCreateBlankReport(t *testing.T) {
	ctx := newReviewContext()
	ctx.subjectDaemon = &dbmodel.Daemon{
		ID: 123,
	}
	report, err := newReport(ctx, "   ").create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that an attempt to create a report with subject daemon ID
// of 0 is not possible.
func TestCreateZeroSubjectDaemonID(t *testing.T) {
	ctx := newReviewContext()
	ctx.subjectDaemon = &dbmodel.Daemon{
		ID: 0,
	}

	report, err := newReport(ctx, "new report").create()
	require.Error(t, err)
	require.Nil(t, report)
}

// Test that an attempt to create a report with referenced daemon
// ID of 0 is not possible.
func TestCreateZeroReferencedDaemonID(t *testing.T) {
	ctx := newReviewContext()
	ctx.subjectDaemon = &dbmodel.Daemon{
		ID: 123,
	}
	referencedDaemon := &dbmodel.Daemon{
		ID: 0,
	}
	report, err := newReport(ctx, "new report").
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
	ctx := newReviewContext()
	ctx.subjectDaemon = daemons[1]

	report, err := newReport(ctx, "new report").
		referencingDaemon(daemons[0]).
		referencingDaemon(daemons[1]).
		referencingDaemon(daemons[2]).
		create()
	require.Error(t, err)
	require.Nil(t, report)
}
