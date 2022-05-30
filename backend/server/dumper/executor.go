package dumper

import (
	"time"

	"isc.org/stork/server/dumper/dump"
)

// Summary of the dump process execution.
// It is the output of the main execution function.
// It contains the time of the execution and results
// of the execution of each dump.
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

// Construct the execution summary object.
// It can be created after the execution (by passing the steps to
// this constructor) or before the execution (append steps to
// the Steps slice member).
func newExecutionSummary(steps ...*executionSummaryStep) *executionSummary {
	return &executionSummary{
		Timestamp: time.Now().UTC(),
		Steps:     steps,
	}
}

// Extract only successfully finished dumps. The dump has a success
// status if no error occurs.
func (s *executionSummary) getSuccessfulDumps() []dump.Dump {
	dumps := make([]dump.Dump, 0)
	for _, step := range s.Steps {
		if step.isSuccess() {
			dumps = append(dumps, step.Dump)
		}
	}
	return dumps
}

// simplify the execution summary to the serializable form.
func (s *executionSummary) simplify() *executionSummarySimplified {
	steps := []*executionSummaryStepSimplified{}

	for _, source := range s.Steps {
		steps = append(steps, source.simplify())
	}

	return &executionSummarySimplified{
		Timestamp: s.Timestamp.Format(time.RFC3339),
		Steps:     steps,
	}
}

// Append summary dump to the steps.
// The summary is included in the dump because contains
// a list of the executed steps and all occurred errors.
// It is a specialized function that creates the summary
// dump and appends it to the steps slice, but also ensures
// that the summary dump contains itself on the executed
// step internal list.
func (s *executionSummary) appendSummaryDump() {
	dumpSummaryArtifact := dump.NewBasicStructArtifact(
		"executed-steps", nil,
	)

	dumpSummary := dump.NewBasicDump(
		"summary",
		dumpSummaryArtifact,
	)

	s.Steps = append(s.Steps, newExecutionSummaryStep(dumpSummary, nil))
	dumpSummaryArtifact.SetStruct(s.simplify())
}

// Construct a new execution summary step instance.
func newExecutionSummaryStep(dump dump.Dump, err error) *executionSummaryStep {
	return &executionSummaryStep{
		Dump:  dump,
		Error: err,
	}
}

// Specifies that the step has no error (the dump was executed correctly).
func (s *executionSummaryStep) isSuccess() bool {
	return s.Error == nil
}

// simplify the execution summary step to the serializable form.
func (s *executionSummaryStep) simplify() *executionSummaryStepSimplified {
	artifactNames := []string{}
	for i := 0; i < s.Dump.GetArtifactsNumber(); i++ {
		artifact := s.Dump.GetArtifact(i)
		artifactNames = append(
			artifactNames,
			artifact.GetName()+artifact.GetExtension(),
		)
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

	// Add the summary to the steps slice. The summary
	// will be included with other dumped data.
	summary.appendSummaryDump()

	return summary
}
