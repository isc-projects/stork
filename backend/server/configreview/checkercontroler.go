package configreview

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Represents a state of config checker managed by the config checker controller.
// Checker for a given condition can be enabled or disabled or inherit the
// state from the higher order rule.
type CheckerState int

const (
	CheckerStateInherit  CheckerState = iota
	CheckerStateDisabled CheckerState = iota
	CheckerStateEnabled  CheckerState = iota
)

// String representation of the checker state.
func (s CheckerState) String() string {
	switch s {
	case CheckerStateDisabled:
		return "disabled"
	case CheckerStateEnabled:
		return "enabled"
	case CheckerStateInherit:
		return "inherit"
	}
	log.WithField("state", fmt.Sprintf("%d", s)).Error("Unknown checker state")
	return "unknown"
}

// Represents a configuration checker controller. It manages the enable or
// disable states of checkers for given conditions, e.g., only for a specific
// daemon, selector, or globally.
// The checkers are enabled by default.
type checkerController interface {
	setGlobalState(checkerName string, state CheckerState)
	getGlobalState(checkerName string) bool
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
// The checkers are enabled by default.
func (c checkerControllerImpl) getGlobalState(checkerName string) bool {
	enabled, ok := c.globalStates[checkerName]
	if !ok {
		return true
	}
	return enabled
}

// Sets the global state for a given checker.
func (c checkerControllerImpl) setGlobalState(checkerName string, state CheckerState) {
	// Resets to default
	if state == CheckerStateInherit {
		delete(c.globalStates, checkerName)
	} else {
		c.globalStates[checkerName] = state == CheckerStateEnabled
	}
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

// Returns a checker state assigned with a given daemon.
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
