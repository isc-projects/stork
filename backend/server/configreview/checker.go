package configreview

// Represents a configuration checker. It includes a checker name,
// triggers which can activate this checker and the pointer to the
// function implementing the checker.
type checker struct {
	name     string
	triggers Triggers
	checkFn  func(*ReviewContext) (*Report, error)
}

// Represents current metadata of the configuration checker. It includes a name,
// triggers, selectors on which the checker was registered, and enable state.
type CheckerMetadata struct {
	Name      string
	Triggers  Triggers
	Selectors DispatchGroupSelectors
	Enabled   bool
	State     CheckerState
}

func newCheckerMetadata(name string, triggers Triggers, selectors DispatchGroupSelectors, enabled bool, state CheckerState) *CheckerMetadata {
	return &CheckerMetadata{
		Name:      name,
		Triggers:  triggers,
		Selectors: selectors,
		Enabled:   enabled,
		State:     state,
	}
}
