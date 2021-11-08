package dumper

import (
	"encoding/json"
	"time"

	dumperdumps "isc.org/stork/server/dumper/dumps"
)

// Summary of the dump process execution.
type executionSummary struct {
	Timestamp time.Time
	Steps     []*executionSummaryStep
}

func newExecutionSummary() *executionSummary {
	return &executionSummary{
		Timestamp: time.Now().UTC(),
		Steps:     make([]*executionSummaryStep, 0),
	}
}

// Extract only successfully finished dumps. The dump has a success
// status if no error occurs.
func (s *executionSummary) GetSuccessDumps() []dumperdumps.Dump {
	dumps := make([]dumperdumps.Dump, 0)
	for _, step := range s.Steps {
		if step.IsSuccess() {
			dumps = append(dumps, step.Dump)
		}
	}
	return dumps
}

// Single dump execution entry. It contains the dump object and related
// error object (or nil if no error occurs).
type executionSummaryStep struct {
	Dump  dumperdumps.Dump
	Error error
}

func newExecutionSummaryStep(dump dumperdumps.Dump, err error) *executionSummaryStep {
	return &executionSummaryStep{
		Dump:  dump,
		Error: err,
	}
}

// Specifies that has no error.
func (s *executionSummaryStep) IsSuccess() bool {
	return s.Error == nil
}

// Custom serialization for the summary. It contains
// the dump artifacts, but we want to serialize the status
// of each task.
func (es executionSummary) MarshalJSON() ([]byte, error) {
	type summaryStepInternal struct {
		Name      string
		Error     error
		Status    string
		Artifacts []string
	}

	type summaryInternal struct {
		Timestamp string
		Steps     []*summaryStepInternal
	}

	var steps []*summaryStepInternal

	for _, source := range es.Steps {
		var artifactNames []string
		for i := 0; i < source.Dump.NumberOfArtifacts(); i++ {
			artifactNames = append(artifactNames, source.Dump.GetArtifact(i).Name())
		}

		status := "SUCCESS"
		if source.Error != nil {
			status = "ERROR"
		}

		steps = append(steps, &summaryStepInternal{
			Name:      source.Dump.Name(),
			Error:     source.Error,
			Artifacts: artifactNames,
			Status:    status,
		})
	}

	summary := summaryInternal{
		Timestamp: es.Timestamp.Format("2006-01-02T15:04:05"),
		Steps:     steps,
	}

	return json.Marshal(summary)
}

// Execute the dump process. Besides the provided dumps the
// result will contain one more dump with the dump summary.
func execute(dumps []dumperdumps.Dump) *executionSummary {
	summary := newExecutionSummary()

	for _, dump := range dumps {
		err := dump.Execute()
		step := newExecutionSummaryStep(dump, err)
		summary.Steps = append(summary.Steps, step)
	}

	dumpSummary := dumperdumps.NewBasicDump(
		"summary",
		dumperdumps.NewBasicStructArtifact(
			"executed-steps", summary,
		),
	)
	summary.Steps = append(summary.Steps, newExecutionSummaryStep(dumpSummary, nil))

	return summary
}
