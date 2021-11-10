package dumper

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/dumper/dumps"
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
	dump := dumps.NewBasicDump("foo")
	err := errors.New("bar")

	// Act
	step := newExecutionSummaryStep(dump, err)

	// Asset
	require.EqualValues(t, "foo", step.Dump.Name())
	require.Error(t, step.Error)
}

// Test that the execution summary is properly constructed with steps.
func TestConstructExecutionSummaryWithSteps(t *testing.T) {
	// Act
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dumps.NewBasicDump("foo"), nil,
		),
		newExecutionSummaryStep(
			dumps.NewBasicDump("bar"), errors.New("bar"),
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
	expectedSuccess := successStep.IsSuccess()
	expectedFail := failedStep.IsSuccess()

	// Assert
	require.True(t, expectedSuccess)
	require.False(t, expectedFail)
}

// Test that the successful dumps are extracted.
func TestGetSuccessfulDumps(t *testing.T) {
	// Arrange
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dumps.NewBasicDump("foo"),
			nil,
		),
		newExecutionSummaryStep(
			dumps.NewBasicDump("bar"),
			errors.New("bar"),
		),
		newExecutionSummaryStep(
			dumps.NewBasicDump("baz"),
			errors.New("baz"),
		),
		newExecutionSummaryStep(
			dumps.NewBasicDump("boz"),
			nil,
		),
	)

	// Act
	dumps := summary.GetSuccessfulDumps()

	// Assert
	require.Len(t, dumps, 2)
	require.EqualValues(t, "foo", dumps[0].Name())
	require.EqualValues(t, "boz", dumps[1].Name())
}

// Test that the execution step without an error is simplified as expected.
func TestSimplifySuccessExecutionStep(t *testing.T) {
	// Arrange
	step := newExecutionSummaryStep(
		dumps.NewBasicDump("foo",
			dumps.NewBasicArtifact("alfa"),
			dumps.NewBasicArtifact("beta"),
			dumps.NewBasicArtifact("gamma"),
		),
		nil,
	)

	// Act
	simplify := step.Simplify()

	// Assert
	require.EqualValues(t, "foo", simplify.Name)
	require.NoError(t, simplify.Error)
	require.EqualValues(t, []string{"alfa", "beta", "gamma"}, simplify.Artifacts)
	require.EqualValues(t, "SUCCESS", simplify.Status)
}

// Test that the execution step with an error is simplified as expected.
func TestSimplifyFailedExecutionStep(t *testing.T) {
	// Arrange
	step := newExecutionSummaryStep(
		dumps.NewBasicDump("foo",
			dumps.NewBasicArtifact("alfa"),
			dumps.NewBasicArtifact("beta"),
			dumps.NewBasicArtifact("gamma"),
		),
		errors.New("foo"),
	)

	// Act
	simplify := step.Simplify()

	// Assert
	require.EqualValues(t, "foo", simplify.Name)
	require.Error(t, simplify.Error)
	require.EqualValues(t, []string{"alfa", "beta", "gamma"}, simplify.Artifacts)
	require.EqualValues(t, "FAIL", simplify.Status)
}

// Test that the execution summary is simplified properly.
func TestSimplifyExecutionSummary(t *testing.T) {
	// Arrange
	summary := newExecutionSummary(
		newExecutionSummaryStep(
			dumps.NewBasicDump("foo",
				dumps.NewBasicArtifact("alfa"),
				dumps.NewBasicArtifact("beta"),
				dumps.NewBasicArtifact("gamma"),
			),
			errors.New("foo"),
		),
	)

	// Act
	simplified := summary.Simplify()
	actualTimestamp, err := time.Parse("2006-01-02T15:04:05 UTC", simplified.Timestamp)

	// Assert
	require.NoError(t, err)
	timeDelta := summary.Timestamp.Sub(actualTimestamp)
	require.LessOrEqual(t, timeDelta.Seconds(), float64(1))
	require.Len(t, simplified.Steps, 1)
}

// Mock dump - only for test purposes.
type mockDump struct {
	dumps.Dump
	err       error
	callCount int
}

func newMockDump(name string, err error) *mockDump {
	return &mockDump{
		dumps.NewBasicDump(name),
		err,
		0,
	}
}

func (d *mockDump) Execute() error {
	d.callCount++
	return d.err
}

// Test that the dumps are executed properly.
func TestExecuteDumps(t *testing.T) {
	// Arrange
	successMock := newMockDump("foo", nil)
	failedMock := newMockDump("foobar", errors.New("fail"))

	dumps := []dumps.Dump{
		successMock,
		dumps.NewBasicDump("bar", dumps.NewBasicArtifact("bir")),
		dumps.NewBasicDump("baz", dumps.NewBasicArtifact("buz"), dumps.NewBasicArtifact("bez")),
		failedMock,
	}

	// Act
	summary := executeDumps(dumps)

	// Assert
	require.EqualValues(t, successMock.callCount, 1)
	require.EqualValues(t, failedMock.callCount, 1)

	require.Len(t, summary.Steps, 5)
	require.EqualValues(t, "bar", summary.Steps[1].Dump.Name())
	require.NoError(t, summary.Steps[1].Error)
	require.EqualValues(t, "foobar", summary.Steps[3].Dump.Name())
	require.Error(t, summary.Steps[3].Error)
}

// Test that the dump execution produces the proper summary dump.
func TestExecuteDumpProducesSummaryDump(t *testing.T) {
	// Arrange
	summary := executeDumps([]dumps.Dump{
		dumps.NewBasicDump("baz", dumps.NewBasicArtifact("buz"), dumps.NewBasicArtifact("bez")),
		newMockDump("bar", errors.New("fail")),
	})

	// Act
	summaryStep := summary.Steps[2]
	summaryArtifact := summaryStep.Dump.GetArtifact(0)
	summaryObject := summaryArtifact.(*dumps.BasicStructArtifact)
	simplifySummary, ok := summaryObject.GetStruct().(*executionSummarySimplify)

	// Assert
	require.True(t, ok)
	require.EqualValues(t, 1, summaryStep.Dump.NumberOfArtifacts())
	require.Len(t, simplifySummary.Steps, 3)
}
