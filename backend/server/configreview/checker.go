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
// The checker metadata is valid only for a specific daemon (or globally).
// It affects the selector list and the state. The enabled property combines
// the daemon state and the global one. It means that for CheckerStateEnabled,
// the enabled property is always true, and for CheckerStateDisabled, it's
// always false, but for CheckerStateInherit, it may be true or false.
type CheckerMetadata struct {
	Name            string
	Triggers        Triggers
	Selectors       DispatchGroupSelectors
	GloballyEnabled bool
	State           CheckerState
}

// Constructs the checker metadata.
func newCheckerMetadata(name string, triggers Triggers, selectors DispatchGroupSelectors, globallyEnabled bool, state CheckerState) *CheckerMetadata {
	return &CheckerMetadata{
		Name:            name,
		Triggers:        triggers,
		Selectors:       selectors,
		GloballyEnabled: globallyEnabled,
		State:           state,
	}
}
