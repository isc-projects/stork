package configreview

import "github.com/pkg/errors"

// Represents a state of config checker managed by the config checker controller.
// Checker for a given condition can be enabled or disabled or inherit the
// state from the higher order rule.
type CheckerState string

// Valid checker states.
const (
	CheckerStateInherit  CheckerState = "inherit"
	CheckerStateDisabled CheckerState = "disabled"
	CheckerStateEnabled  CheckerState = "enabled"
)

// Represents a configuration checker controller. It manages the enable or
// disable states of checkers for given conditions, e.g., only for a specific
// daemon, selector, or globally.
// The checkers are enabled by default.
type checkerController interface {
	setGlobalState(checkerName string, state CheckerState) error
	getGlobalState(checkerName string) CheckerState
	setStateForDaemon(daemonID int64, checkerName string, state CheckerState)
	isCheckerEnabledForDaemon(daemonID int64, checkerName string) bool
	getStateForDaemon(daemonID int64, checkerName string) CheckerState
}

// Implementation of the checker controller interface.
type checkerControllerImpl struct {
	globalStates map[string]bool
	daemonStates map[int64]map[string]bool
}

// Constructs the checker controller object.
func newCheckerController() checkerController {
	return &checkerControllerImpl{
		globalStates: make(map[string]bool),
		daemonStates: make(map[int64]map[string]bool),
	}
}

// Returns true if a checker with the given name is enabled.
// The checkers are enabled by default. The global state is never inherited.
func (c checkerControllerImpl) getGlobalState(checkerName string) CheckerState {
	enabled, ok := c.globalStates[checkerName]
	if !ok || enabled {
		return CheckerStateEnabled
	}
	return CheckerStateDisabled
}

// Sets the global state for a given checker. The inherit state isn't accepted.
func (c checkerControllerImpl) setGlobalState(checkerName string, state CheckerState) error {
	if state == CheckerStateInherit {
		return errors.Errorf("the global state cannot be inherit")
	}

	c.globalStates[checkerName] = state == CheckerStateEnabled
	return nil
}

// Sets the state of config checker for a specific daemon.
func (c checkerControllerImpl) setStateForDaemon(daemonID int64, checkerName string, state CheckerState) {
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
// daemon state with a global one. The checkers are enabled by default.
func (c checkerControllerImpl) isCheckerEnabledForDaemon(daemonID int64, checkerName string) bool {
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

// Returns a checker state assigned to a given daemon.
func (c checkerControllerImpl) getStateForDaemon(daemonID int64, checkerName string) CheckerState {
	if _, ok := c.daemonStates[daemonID]; ok {
		if enabled, ok := c.daemonStates[daemonID][checkerName]; ok {
			if enabled {
				return CheckerStateEnabled
			}
			return CheckerStateDisabled
		}
	}

	return CheckerStateInherit
}
