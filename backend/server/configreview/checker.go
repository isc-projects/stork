package configreview

// Represents a configuration checker. It includes a checker name,
// triggers which can activate this checker and the pointer to the
// function implementing the checker.
type checker struct {
	name     string
	triggers Triggers
	checkFn  func(*ReviewContext) (*Report, error)
}
