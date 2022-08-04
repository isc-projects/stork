package configreview

// Represents a state of config checker managed by the config checker controller.
// Checker for a given condition can be enabled or disabled or inherit the
// state from the higher order rule.
type CheckerState int

const (
	CheckerStateInherit  CheckerState = iota
	CheckerStateDisabled CheckerState = iota
	CheckerStateEnabled  CheckerState = iota
)

// Represents a configuration checker controller. It manages the enable or
// disable states of checkers for given conditions, e.g., only for a specific
// daemon, selector, or globally.
// The checkers are enabled by default.
type checkerController interface {
	SetStateGlobally(checkerName string, enabled bool)
	SetStateForDaemon(daemonID int64, checkerName string, state CheckerState)
	IsCheckerEnabledForDaemon(daemonID int64, checkerName string) bool
}

// Implementation of the checker controller interface.
type checkerControllerImpl struct {
	globalStates map[string]bool
	daemonStates map[int64]map[string]bool
}

// Constructs the checker controller object.
func NewCheckerController() checkerController {
	return &checkerControllerImpl{
		globalStates: make(map[string]bool),
		daemonStates: make(map[int64]map[string]bool),
	}
}

// Sets the global state for a given checker.
func (c checkerControllerImpl) SetStateGlobally(checkerName string, enabled bool) {
	c.globalStates[checkerName] = enabled
}

// Sets the state of config checker for a specific daemon.
func (c checkerControllerImpl) SetStateForDaemon(daemonID int64, checkerName string, state CheckerState) {
	if _, ok := c.daemonStates[daemonID]; !ok {
		c.daemonStates[daemonID] = make(map[string]bool)
	}

	if state == CheckerStateInherit {
		delete(c.daemonStates[daemonID], checkerName)
	} else {
		c.daemonStates[daemonID][checkerName] = state == CheckerStateEnabled
	}
}

// Lookups for the state of config checker for a given daemon. It combines the
// daemon state with a global one.
func (c checkerControllerImpl) IsCheckerEnabledForDaemon(daemonID int64, checkerName string) bool {
	if _, ok := c.daemonStates[daemonID]; ok {
		if enabled, ok := c.daemonStates[daemonID][checkerName]; ok {
			return enabled
		}
	}
	if enabled, ok := c.globalStates[checkerName]; ok {
		return enabled
	}
	return true
}
