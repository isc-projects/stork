package dumper

import (
	"time"

	"isc.org/stork/server/dumper/dump"
)

// Summary of the dump process execution.
type executionSummary struct {
	Timestamp time.Time
	Steps     []*executionSummaryStep
}

// Single dump execution entry. It contains the dump object and related
// error object (or nil if no error occurs).
type executionSummaryStep struct {
	Dump  dump.Dump
	Error error
}

// Simplified representation of the summary
// to use in the dump.
type executionSummarySimplified struct {
	Timestamp string
	Steps     []*executionSummaryStepSimplified
}

// Simplified representation of the summary step
// to use in the dump export.
type executionSummaryStepSimplified struct {
	Name      string
	Error     error `json:",omitempty"`
	Status    string
	Artifacts []string
}

func newExecutionSummary(steps ...*executionSummaryStep) *executionSummary {
	return &executionSummary{
		Timestamp: time.Now().UTC(),
		Steps:     steps,
	}
}

// Extract only successfully finished dumps. The dump has a success
// status if no error occurs.
func (s *executionSummary) GetSuccessfulDumps() []dump.Dump {
	dumps := make([]dump.Dump, 0)
	for _, step := range s.Steps {
		if step.IsSuccess() {
			dumps = append(dumps, step.Dump)
		}
	}
	return dumps
}

// Simplify the execution summary to the serializable form.
func (s *executionSummary) Simplify() *executionSummarySimplified {
	var steps []*executionSummaryStepSimplified

	for _, source := range s.Steps {
		steps = append(steps, source.Simplify())
	}

	return &executionSummarySimplified{
		Timestamp: s.Timestamp.Format(time.RFC3339),
		Steps:     steps,
	}
}

// Append summary dump to the steps.
func (s *executionSummary) appendSummaryDump() {
	dumpSummaryArtifact := dump.NewBasicStructArtifact(
		"executed-steps", nil,
	)

	dumpSummary := dump.NewBasicDump(
		"summary",
		dumpSummaryArtifact,
	)

	s.Steps = append(s.Steps, newExecutionSummaryStep(dumpSummary, nil))
	dumpSummaryArtifact.SetStruct(s.Simplify())
}

// Construct a new execution summary step instance.
func newExecutionSummaryStep(dump dump.Dump, err error) *executionSummaryStep {
	return &executionSummaryStep{
		Dump:  dump,
		Error: err,
	}
}

// Specifies that has no error.
func (s *executionSummaryStep) IsSuccess() bool {
	return s.Error == nil
}

// Simplify the execution summary step to the serializable form.
func (s *executionSummaryStep) Simplify() *executionSummaryStepSimplified {
	var artifactNames []string
	for i := 0; i < s.Dump.GetArtifactsNumber(); i++ {
		artifactNames = append(artifactNames, s.Dump.GetArtifact(i).GetName())
	}

	status := "SUCCESS"
	if s.Error != nil {
		status = "FAIL"
	}

	return &executionSummaryStepSimplified{
		Name:      s.Dump.GetName(),
		Error:     s.Error,
		Artifacts: artifactNames,
		Status:    status,
	}
}

// Execute the dump process. Besides the provided dumps the
// result will contain one more dump with the dump summary.
func executeDumps(dumps []dump.Dump) *executionSummary {
	summary := newExecutionSummary()

	for _, dump := range dumps {
		err := dump.Execute()
		step := newExecutionSummaryStep(dump, err)
		summary.Steps = append(summary.Steps, step)
	}

	summary.appendSummaryDump()

	return summary
}
