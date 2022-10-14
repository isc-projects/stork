package hooksutil

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Mock callout interfaces.
type mockCalloutFoo interface {
	Foo() int
}

type mockCalloutBar interface {
	Bar() bool
}

type mockCalloutFooBar interface {
	Foo() int
	Bar() bool
}

// Foo mock callout implementation.
type mockCalloutFooImpl struct {
	fooCount int
	// The call count including calls from all mock instances.
	fooTotalCount int
}

// The Foo call counter shared between all mock instances.
var sharedFooCount int //nolint:gochecknoglobals

// Counts the call count.
func (c *mockCalloutFooImpl) Foo() int {
	c.fooCount++
	sharedFooCount++
	c.fooTotalCount = sharedFooCount
	return c.fooTotalCount
}

// Constructs an instance of the mock callout implementation.
func newMockCalloutFoo() *mockCalloutFooImpl {
	return &mockCalloutFooImpl{
		fooCount: 0,
	}
}

// Mock callout implementation that implement the mock callout interface
// and io.Closer.
type mockCalloutFooClosableImpl struct {
	mockCalloutFooImpl

	closeCount int
	closeErr   error
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutFooClosableImpl) Close() error {
	c.closeCount++
	return c.closeErr
}

// Constructs the mock callout. Accepts an error returned by the Close method.
func newClosableCallout(closeErr error) *mockCalloutFooClosableImpl {
	return &mockCalloutFooClosableImpl{
		mockCalloutFooImpl: *newMockCalloutFoo(),
		closeCount:         0,
		closeErr:           closeErr,
	}
}

// Bar mock implementation.
type mockCalloutBarImpl struct {
	barCount int
}

// Counts the calls. Return parity of an actual value.
func (m *mockCalloutBarImpl) Bar() bool {
	m.barCount++
	return m.barCount%2 == 0
}

// Constructs the Bar mock.
func newMockCalloutBar() *mockCalloutBarImpl {
	return &mockCalloutBarImpl{}
}

// FooBar mock implementation.
type mockCalloutFooBarImpl struct {
	mockCalloutFooImpl
	mockCalloutBarImpl
}

// Constructs the FooBar mock.
func newMockCalloutFooBar() *mockCalloutFooBarImpl {
	return &mockCalloutFooBarImpl{}
}

// Test that the hook executor is constructed properly.
func TestNewHookExecutor(t *testing.T) {
	// Arrange & Act
	emptyExecutor := NewHookExecutor([]reflect.Type{})
	nilExecutor := NewHookExecutor(nil)
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutFoo)(nil)).Elem(),
	})

	// Assert
	require.NotNil(t, emptyExecutor)
	require.NotNil(t, nilExecutor)
	require.NotNil(t, executor)

	require.Contains(t, executor.registeredCallouts, reflect.TypeOf((*mockCalloutFoo)(nil)).Elem())
}

// Test that the hook executor constructor panics on an invalid type (it's a
// programming bug).
func TestNewHookExecutorInvalidType(t *testing.T) {
	// Arrange
	// Missing .Elem() call
	invalidType := reflect.TypeOf((*mockCalloutFoo)(nil))

	// Assert
	require.Panics(t, func() {
		// Act
		_ = NewHookExecutor([]reflect.Type{invalidType})
	})
}

// Test that the supported callouts object is registered properly.
func TestRegisterSupportedCallouts(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	// Act
	executor.RegisterCallouts(newMockCalloutFoo())

	// Assert
	require.NotEmpty(t, executor.registeredCallouts[calloutType])
}

// Test that the unsupported callouts object is not registered.
func TestRegisterUnsupportedCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	executor.RegisterCallouts(newMockCalloutFoo())

	// Assert
	require.Empty(t, executor.registeredCallouts)
}

// Test that all callouts are unregistered.
func TestUnregisterAllCallouts(t *testing.T) {
	// Arrange
	callout := newClosableCallout(nil)
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
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
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})
	executor.RegisterCallouts(newMockCalloutFoo())

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

	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
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

// Test that the registered callout is detected as registered.
func TestHasRegisteredForRegisteredCallout(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	executor.RegisterCallouts(newMockCalloutFoo())

	// Act
	isRegistered := executor.HasRegistered(calloutType)

	// Assert
	require.True(t, isRegistered)
}

// Test that the non-registered callout is non detected as registered.
func TestHasRegisteredForNonRegisteredCallout(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	// Act
	isRegistered := executor.HasRegistered(calloutType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the unsupported callout is not detected as registered.
func TestHasRegisteredForUnsupportedCallout(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	isRegistered := executor.HasRegistered(calloutType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the callout points are called in the sequential order properly.
func TestCallSequential(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	fooMocks := []*mockCalloutFooImpl{
		newMockCalloutFoo(),
		newMockCalloutFoo(),
		newMockCalloutFoo(),
	}
	barMock := newMockCalloutBar()
	fooBarMock := newMockCalloutFooBar()

	for _, mock := range fooMocks {
		executor.RegisterCallouts(mock)
	}
	executor.RegisterCallouts(barMock)
	executor.RegisterCallouts(fooBarMock)

	// Act
	results := CallSequential(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	// 1. One result for each callout object.
	require.Len(t, results, len(fooMocks)+1)

	for i, mock := range fooMocks {
		result := results[i]

		// 2. Has expected output.
		require.EqualValues(t, mock.fooTotalCount, result)
		// 3. The callout object was called exactly once.
		require.EqualValues(t, 1, mock.fooCount)

		if i != 0 {
			previousMock := fooMocks[i-1]
			// 4. The callout objects were executed in an expected order.
			require.EqualValues(t, previousMock.fooTotalCount, mock.fooTotalCount-1)
		}
	}

	// 5. FooBar mock should be called too.
	require.EqualValues(t, 1, fooBarMock.fooCount)
	require.EqualValues(t, fooBarMock.fooTotalCount, results[len(fooMocks)])
	require.Zero(t, fooBarMock.barCount)

	// 6. Bar mock shouldn't be called.
	require.Zero(t, barMock.barCount)
}

// Test that the callout point is executed properly if exactly one callout object
// was registered.
func TestCallSingleForOneRegisteredCallout(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	mock := newMockCalloutFoo()

	executor.RegisterCallouts(mock)

	// Act
	result := CallSingle(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	require.EqualValues(t, 1, mock.fooCount)
	require.EqualValues(t, mock.fooTotalCount, result)
}

// Test that only the first callout point is executed if more than one callout
// object was registered.
func TestCallSingleForManyRegisteredCallouts(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	mocks := []*mockCalloutFooImpl{
		newMockCalloutFoo(),
		newMockCalloutFoo(),
		newMockCalloutFoo(),
	}

	for _, mock := range mocks {
		executor.RegisterCallouts(mock)
	}

	// Act
	result := CallSingle(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	require.EqualValues(t, 1, mocks[0].fooCount)
	require.EqualValues(t, mocks[0].fooTotalCount, result)

	for i := 1; i < len(mocks); i++ {
		mock := mocks[i]

		require.Zero(t, mock.fooCount)
	}
}

// Test that the default result is returned if no callout object was
// registered.
func TestCallSingleForNoRegisteredCallouts(t *testing.T) {
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	// Act
	result := CallSingle(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	require.Zero(t, result)
}
