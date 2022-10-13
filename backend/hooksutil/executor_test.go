package hooksutil

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Mock callout interface. Specifies nothing.
type mockCallout interface{}

// Mock callout implementation that implement the mock callout interface
// and io.Closer.
type mockCalloutClosable struct {
	closeCount int
	closeErr   error
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutClosable) Close() error {
	c.closeCount++
	return c.closeErr
}

// Constructs the mock callout. Accepts an error returned by the Close method.
func newClosableCallout(closeErr error) *mockCalloutClosable {
	return &mockCalloutClosable{
		closeCount: 0,
		closeErr:   closeErr,
	}
}

// Test that the hook executor is constructed properly.
func TestNewHookExecutor(t *testing.T) {
	// Arrange & Act
	emptyExecutor := NewHookExecutor([]reflect.Type{})
	nilExecutor := NewHookExecutor(nil)
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCallout)(nil)).Elem(),
	})

	// Assert
	require.NotNil(t, emptyExecutor)
	require.NotNil(t, nilExecutor)
	require.NotNil(t, executor)

	require.Contains(t, executor.registeredCallouts, reflect.TypeOf((*mockCallout)(nil)).Elem())
}

// Test that the hook executor constructor panics on an invalid type (it's a
// programming bug).
func TestNewHookExecutorInvalidType(t *testing.T) {
	// Arrange
	// Missing .Elem() call
	invalidType := reflect.TypeOf((*mockCallout)(nil))

	// Assert
	require.Panics(t, func() {
		// Act
		_ = NewHookExecutor([]reflect.Type{invalidType})
	})
}

// Test that the supported callouts object is registered properly.
func TestRegisterSupportedCallouts(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCallout)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	// Act
	executor.RegisterCallouts(&struct{}{})

	// Assert
	require.NotEmpty(t, executor.registeredCallouts[calloutType])
}

// Test that the unsupported callouts object is not registered.
func TestRegisterUnsupportedCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	executor.RegisterCallouts(&struct{}{})

	// Assert
	require.Empty(t, executor.registeredCallouts)
}

// Test that all callouts are unregistered.
func TestUnregisterAllCallouts(t *testing.T) {
	// Arrange
	callout := newClosableCallout(nil)
	calloutType := reflect.TypeOf((*mockCallout)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})
	executor.RegisterCallouts(callout)

	// Act
	errs := executor.UnregisterAllCallouts()

	// Assert
	require.Empty(t, executor.registeredCallouts)
	require.EqualValues(t, 1, callout.closeCount)
	require.Empty(t, errs)
}

// Test that callout without Close method causes no error.
// Note that the Close function is mandatory in the standard flow.
func TestUnregisterCalloutWithoutClose(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCallout)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})
	executor.RegisterCallouts(&struct{}{})

	// Act
	errs := executor.UnregisterAllCallouts()

	// Assert
	require.Empty(t, executor.registeredCallouts)
	require.Empty(t, errs)
}

// Test that if one callout object returns an error, other are unregistered
// properly.
func TestUnregisterAllCalloutsWithError(t *testing.T) {
	// Arrange
	successCallout := newClosableCallout(nil)
	failedCallout := newClosableCallout(errors.New("Close failed"))

	calloutType := reflect.TypeOf((*mockCallout)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	executor.RegisterCallouts(successCallout)
	executor.RegisterCallouts(failedCallout)

	// Act
	errs := executor.UnregisterAllCallouts()

	// Assert
	require.Len(t, errs, 1)
	require.EqualValues(t, 1, successCallout.closeCount)
	require.EqualValues(t, 1, failedCallout.closeCount)
}
