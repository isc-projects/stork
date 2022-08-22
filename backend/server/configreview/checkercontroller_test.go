package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the checker controller is constructed properly.
func TestNewCheckerController(t *testing.T) {
	// Act
	controller := newCheckerController()

	// Assert
	require.NotNil(t, controller)
}

// Test that the checker state is returned properly.
func TestGetGlobalState(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.setGlobalState("foo", CheckerStateEnabled)
	controller.setGlobalState("bar", CheckerStateDisabled)

	// Act
	foo := controller.getGlobalState("foo")
	bar := controller.getGlobalState("bar")
	baz := controller.getGlobalState("baz")

	// Assert
	require.EqualValues(t, CheckerStateEnabled, foo)
	require.EqualValues(t, CheckerStateDisabled, bar)
	require.EqualValues(t, CheckerStateEnabled, baz)
}

// Test that the checker state is set properly.
func TestSetGlobalState(t *testing.T) {
	// Arrange
	controller := newCheckerController()

	// Act
	err1 := controller.setGlobalState("foo", CheckerStateEnabled)
	err2 := controller.setGlobalState("bar", CheckerStateDisabled)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.True(t, controller.isCheckerEnabledForDaemon(0, "foo"))
	require.False(t, controller.isCheckerEnabledForDaemon(0, "bar"))
	require.True(t, controller.isCheckerEnabledForDaemon(0, "baz"))
}

// Test that global state cannot be set to inherit value.
func TestSetGlobalStateInherit(t *testing.T) {
	// Arrange
	controller := newCheckerController()

	// Act
	err := controller.setGlobalState("baz", CheckerStateInherit)

	// Assert
	require.Error(t, err)
}

// Test that the checker state for a specific daemon is set properly.
func TestSetStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()

	// Act
	controller.setStateForDaemon(1, "foo", CheckerStateEnabled)
	controller.setStateForDaemon(2, "bar", CheckerStateDisabled)
	controller.setStateForDaemon(3, "baz", CheckerStateInherit)

	// Assert
	require.True(t, controller.isCheckerEnabledForDaemon(1, "foo"))
	require.False(t, controller.isCheckerEnabledForDaemon(2, "bar"))
	require.True(t, controller.isCheckerEnabledForDaemon(3, "baz"))
}

// Test that the checker state is correctly inherited for a specific daemon.
func TestSetInheritedStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.setGlobalState("foo", CheckerStateEnabled)
	controller.setGlobalState("bar", CheckerStateDisabled)
	controller.setGlobalState("baz", CheckerStateDisabled)

	// Act
	controller.setStateForDaemon(1, "foo", CheckerStateInherit)
	controller.setStateForDaemon(2, "bar", CheckerStateInherit)
	controller.setStateForDaemon(3, "baz", CheckerStateInherit)
	err := controller.setGlobalState("baz", CheckerStateEnabled)

	// Assert
	require.NoError(t, err)
	require.True(t, controller.isCheckerEnabledForDaemon(1, "foo"))
	require.False(t, controller.isCheckerEnabledForDaemon(2, "bar"))
	require.True(t, controller.isCheckerEnabledForDaemon(3, "baz"))
}

// Test that the checker states are merged properly.
func TestIsCheckerEnabledForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.setGlobalState("foo", CheckerStateEnabled)
	controller.setGlobalState("fee", CheckerStateDisabled)
	controller.setGlobalState("bar", CheckerStateEnabled)
	controller.setStateForDaemon(1, "bar", CheckerStateEnabled)
	controller.setGlobalState("baz", CheckerStateEnabled)
	controller.setStateForDaemon(1, "baz", CheckerStateDisabled)
	controller.setGlobalState("biz", CheckerStateDisabled)
	controller.setStateForDaemon(1, "biz", CheckerStateEnabled)
	controller.setGlobalState("boz", CheckerStateDisabled)
	controller.setStateForDaemon(1, "boz", CheckerStateDisabled)

	// Act
	foo := controller.isCheckerEnabledForDaemon(1, "foo")
	fee := controller.isCheckerEnabledForDaemon(1, "fee")
	bar := controller.isCheckerEnabledForDaemon(1, "bar")
	baz := controller.isCheckerEnabledForDaemon(1, "baz")
	biz := controller.isCheckerEnabledForDaemon(1, "biz")
	boz := controller.isCheckerEnabledForDaemon(1, "boz")

	// Assert
	require.True(t, foo)
	require.False(t, fee)
	require.True(t, bar)
	require.False(t, baz)
	require.True(t, biz)
	require.False(t, boz)
}

// Test that own state of config checker is returned properly.
func TestGetCheckerOwnStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.setStateForDaemon(1, "foo", CheckerStateDisabled)
	controller.setStateForDaemon(1, "bar", CheckerStateEnabled)
	controller.setStateForDaemon(1, "baz", CheckerStateInherit)

	// Act
	foo := controller.getStateForDaemon(1, "foo")
	bar := controller.getStateForDaemon(1, "bar")
	baz := controller.getStateForDaemon(1, "baz")
	boz := controller.getStateForDaemon(1, "boz")

	// Assert
	require.EqualValues(t, CheckerStateDisabled, foo)
	require.EqualValues(t, CheckerStateEnabled, bar)
	require.EqualValues(t, CheckerStateInherit, baz)
	require.EqualValues(t, CheckerStateInherit, boz)
}

// Test that the config checker state is serialized to string properly.
func TestConfigCheckerStateToString(t *testing.T) {
	require.EqualValues(t, "disabled", string(CheckerStateDisabled))
	require.EqualValues(t, "enabled", string(CheckerStateEnabled))
	require.EqualValues(t, "inherit", string(CheckerStateInherit))
	require.EqualValues(t, "unknown", string(CheckerState("unknown")))
}
