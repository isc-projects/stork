package configreview

type CheckerState int

const (
	CheckerStateInherit  CheckerState = iota
	CheckerStateDisabled CheckerState = iota
	CheckerStateEnabled  CheckerState = iota
)

// Represents a configuration checker controller. It manages the enable or
// disable states of checkers for given conditions, e.g., only for a specific
// daemon, selector, or globally.
type checkerController interface {
	SetStateGlobally(checkerName string, enabled bool)
	SetStateForDaemon(daemonID int64, checkerName string, state CheckerState)
	IsCheckerEnabledForDaemon(daemonID int64, checkerName string) bool
}
