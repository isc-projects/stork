package dumper

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/dumper/dump"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Test that the execution summary is properly constructed.
func TestConstructExecutionSummary(t *testing.T) {
	// Act
	summary := newExecutionSummary()

	// Assert
	timeDelta := time.Now().UTC().Sub(summary.Timestamp)
	require.LessOrEqual(t, timeDelta.Seconds(), float64(10))
	require.Len(t, summary.Steps, 0)
}

// Test that the execution step summary is properly constructed.
func TestConstructExecutionSummaryStep(t *testing.T) {
	// Arrange
	dump := dump.NewBasicDump("foo")
	err := errors.New("bar")

	// Act
	step := newExecutionSummaryStep(dump, err)

	// Asset
	require.EqualValues(t, "foo", step.Dump.GetName())
	require.Error(t, step.Error)
}

// Test that the execution summary is properly constructed with steps.
func TestConstructExecutionSummaryWithSteps(t *testing.T) {
	// Act
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dump.NewBasicDump("foo"), nil,
		),
		newExecutionSummaryStep(
			dump.NewBasicDump("bar"), errors.New("bar"),
		),
	)

	// Assert
	require.Len(t, summary.Steps, 2)
}

// Test that the execution step returns a correct success status.
func TestExecutionStepIsSuccess(t *testing.T) {
	// Arrange
	successStep := newExecutionSummaryStep(nil, nil)
	failedStep := newExecutionSummaryStep(nil, errors.New("fail"))

	// Act
	expectedSuccess := successStep.isSuccess()
	expectedFail := failedStep.isSuccess()

	// Assert
	require.True(t, expectedSuccess)
	require.False(t, expectedFail)
}

// Test that the successful dumps are extracted.
func TestGetSuccessfulDumps(t *testing.T) {
	// Arrange
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dump.NewBasicDump("foo"),
			nil,
		),
		newExecutionSummaryStep(
			dump.NewBasicDump("bar"),
			errors.New("bar"),
		),
		newExecutionSummaryStep(
			dump.NewBasicDump("baz"),
			errors.New("baz"),
		),
		newExecutionSummaryStep(
			dump.NewBasicDump("boz"),
			nil,
		),
	)

	// Act
	dumps := summary.getSuccessfulDumps()

	// Assert
	require.Len(t, dumps, 2)
	require.EqualValues(t, "foo", dumps[0].GetName())
	require.EqualValues(t, "boz", dumps[1].GetName())
}

// Test that the execution step without an error is simplified as expected.
func TestSimplifySuccessExecutionStep(t *testing.T) {
	// Arrange
	step := newExecutionSummaryStep(
		dump.NewBasicDump("foo",
			dump.NewBasicArtifact("alfa", ".a"),
			dump.NewBasicArtifact("beta", ".b"),
			dump.NewBasicArtifact("gamma", ".g"),
		),
		nil,
	)

	// Act
	simplify := step.simplify()

	// Assert
	require.EqualValues(t, "foo", simplify.Name)
	require.NoError(t, simplify.Error)
	require.EqualValues(t, []string{"alfa.a", "beta.b", "gamma.g"}, simplify.Artifacts)
	require.EqualValues(t, "SUCCESS", simplify.Status)
}

// Test that the execution step with an empty dump is simplified as expected.
func TestSimplifyExecutionStepWithEmptyDump(t *testing.T) {
	// Arrange
	step := newExecutionSummaryStep(dump.NewBasicDump("foo"), nil)

	// Act
	simplify := step.simplify()

	// Assert
	require.EqualValues(t, "foo", simplify.Name)
	require.NoError(t, simplify.Error)
	require.EqualValues(t, "SUCCESS", simplify.Status)
	require.NotNil(t, simplify.Artifacts)
	require.Len(t, simplify.Artifacts, 0)
}

// Test that the execution step with an error is simplified as expected.
func TestSimplifyFailedExecutionStep(t *testing.T) {
	// Arrange
	step := newExecutionSummaryStep(
		dump.NewBasicDump("foo",
			dump.NewBasicArtifact("alfa", ".a"),
			dump.NewBasicArtifact("beta", ".b"),
			dump.NewBasicArtifact("gamma", ".g"),
		),
		errors.New("foo"),
	)

	// Act
	simplify := step.simplify()

	// Assert
	require.EqualValues(t, "foo", simplify.Name)
	require.Error(t, simplify.Error)
	require.EqualValues(t, []string{"alfa.a", "beta.b", "gamma.g"}, simplify.Artifacts)
	require.EqualValues(t, "FAIL", simplify.Status)
}

// Test that the execution summary is simplified properly.
func TestSimplifyExecutionSummary(t *testing.T) {
	// Arrange
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dump.NewBasicDump("foo",
				dump.NewBasicArtifact("alfa", ".a"),
				dump.NewBasicArtifact("beta", ".b"),
				dump.NewBasicArtifact("gamma", ".g"),
			),
			errors.New("foo"),
		),
	)

	// Act
	simplified := summary.simplify()
	actualTimestamp, err := time.Parse(time.RFC3339, simplified.Timestamp)

	// Assert
	require.NoError(t, err)
	timeDelta := summary.Timestamp.Sub(actualTimestamp)
	require.LessOrEqual(t, timeDelta.Seconds(), float64(1))
	require.Len(t, simplified.Steps, 1)
}

// Test that the empty execution summary is simplified properly.
func TestSimplifyEmptyExecutionSummary(t *testing.T) {
	// Arrange
	summary := newExecutionSummary()

	// Act
	simplified := summary.simplify()
	actualTimestamp, err := time.Parse(time.RFC3339, simplified.Timestamp)

	// Assert
	require.NoError(t, err)
	timeDelta := summary.Timestamp.Sub(actualTimestamp)
	require.LessOrEqual(t, timeDelta.Seconds(), float64(1))
	require.NotNil(t, simplified.Steps)
	require.Len(t, simplified.Steps, 0)
}

// Test that the dumps are executed properly.
func TestExecuteDumps(t *testing.T) {
	// Arrange
	successMock := storktest.NewMockDump("foo", nil)
	failedMock := storktest.NewMockDump("foobar", errors.New("fail"))

	dumps := []dump.Dump{
		successMock,
		dump.NewBasicDump("bar", dump.NewBasicArtifact("bir", ".eir")),
		dump.NewBasicDump("baz", dump.NewBasicArtifact("buz", ".euz"),
			dump.NewBasicArtifact("bez", ".eez")),
		failedMock,
	}

	// Act
	summary := executeDumps(dumps)

	// Assert
	require.EqualValues(t, successMock.CallCount, 1)
	require.EqualValues(t, failedMock.CallCount, 1)

	require.Len(t, summary.Steps, 5)
	require.EqualValues(t, "bar", summary.Steps[1].Dump.GetName())
	require.NoError(t, summary.Steps[1].Error)
	require.EqualValues(t, "foobar", summary.Steps[3].Dump.GetName())
	require.Error(t, summary.Steps[3].Error)
}

// Test that the dump execution produces the proper summary dump.
func TestExecuteDumpProducesSummaryDump(t *testing.T) {
	// Arrange
	summary := executeDumps([]dump.Dump{
		dump.NewBasicDump("baz", dump.NewBasicArtifact("buz", ""),
			dump.NewBasicArtifact("bez", "")),
		storktest.NewMockDump("bar", errors.New("fail")),
	})

	// Act
	summaryStep := summary.Steps[2]
	summaryArtifact := summaryStep.Dump.GetArtifact(0)
	summaryObject := summaryArtifact.(*dump.BasicStructArtifact)
	simplifySummary, ok := summaryObject.GetStruct().(*executionSummarySimplified)

	// Assert
	require.True(t, ok)
	require.EqualValues(t, 1, summaryStep.Dump.GetArtifactsNumber())
	require.Len(t, simplifySummary.Steps, 3)
}
