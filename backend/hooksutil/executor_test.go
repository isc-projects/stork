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
	Close() error
}

type mockCalloutBar interface {
	Bar() bool
	Close() error
}

// Foo mock callout implementation.
type mockCalloutFooImpl struct {
	fooCount int
	// The call count including calls from all mock instances.
	fooTotalCount int
	closeCount    int
	closeErr      error
}

// The Foo call counter shared between all mock instances.
var sharedFooCount int //nolint:gochecknoglobals

// Constructs an instance of the mock callout implementation.
func newMockCalloutFoo() *mockCalloutFooImpl {
	return &mockCalloutFooImpl{
		fooCount:      0,
		fooTotalCount: 0,
		closeCount:    0,
		closeErr:      nil,
	}
}

// Counts the call count.
func (c *mockCalloutFooImpl) Foo() int {
	c.fooCount++
	sharedFooCount++
	c.fooTotalCount = sharedFooCount
	return c.fooTotalCount
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutFooImpl) Close() error {
	c.closeCount++
	return c.closeErr
}

// Bar mock implementation.
type mockCalloutBarImpl struct {
	barCount   int
	closeCount int
	closeErr   error
}

// Constructs the Bar mock.
func newMockCalloutBar() *mockCalloutBarImpl {
	return &mockCalloutBarImpl{}
}

// Counts the calls. Return parity of an actual value.
func (c *mockCalloutBarImpl) Bar() bool {
	c.barCount++
	return c.barCount%2 == 0
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutBarImpl) Close() error {
	c.closeCount++
	return c.closeErr
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

func (c *mockCalloutFooBarImpl) Close() error {
	return c.mockCalloutFooImpl.Close()
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

// Test that the supported callout object is registered properly.
func TestRegisterSupportedCallout(t *testing.T) {
	// Arrange
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	// Act
	executor.registerCallout(newMockCalloutFoo())

	// Assert
	require.NotEmpty(t, executor.registeredCallouts[calloutType])
}

// Test that the unsupported callout object is not registered.
func TestRegisterUnsupportedCallout(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	executor.registerCallout(newMockCalloutFoo())

	// Assert
	require.Empty(t, executor.registeredCallouts)
}

// Test that all callouts are unregistering.
func TestUnregisterAllCallouts(t *testing.T) {
	// Arrange
	callout := newMockCalloutFoo()
	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})
	executor.registerCallout(callout)

	// Act
	errs := executor.unregisterAllCallouts()

	// Assert
	require.Empty(t, executor.registeredCallouts)
	require.EqualValues(t, 1, callout.closeCount)
	require.Empty(t, errs)
}

// Test that if one callout object returns an error, other are unregistered
// properly.
func TestUnregisterAllCalloutsWithError(t *testing.T) {
	// Arrange
	successCallout := newMockCalloutFoo()
	failedCallout := newMockCalloutFoo()
	failedCallout.closeErr = errors.New("Close failed")

	calloutType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		calloutType,
	})

	executor.registerCallout(successCallout)
	executor.registerCallout(failedCallout)

	// Act
	errs := executor.unregisterAllCallouts()

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

	executor.registerCallout(newMockCalloutFoo())

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

// Test that the supported callout types are returned properly.
func TestGetSupportedCalloutTypes(t *testing.T) {
	// Arrange
	fooType := reflect.TypeOf((*mockCalloutFoo)(nil)).Elem()
	barType := reflect.TypeOf((*mockCalloutBar)(nil)).Elem()

	executor := NewHookExecutor([]reflect.Type{
		fooType,
		barType,
	})

	// Act
	supportedTypes := executor.GetSupportedCalloutTypes()

	// Assert
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, fooType)
	require.Contains(t, supportedTypes, barType)
}

// Test that the callout points are called in the sequential order properly.
func TestCallSequential(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutBar)(nil)).Elem(),
	})

	fooMocks := []*mockCalloutFooImpl{
		newMockCalloutFoo(),
		newMockCalloutFoo(),
		newMockCalloutFoo(),
	}
	barMock := newMockCalloutBar()
	fooBarMock := newMockCalloutFooBar()

	for _, mock := range fooMocks {
		executor.registerCallout(mock)
	}
	executor.registerCallout(barMock)
	executor.registerCallout(fooBarMock)

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
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutBar)(nil)).Elem(),
	})

	fooMock := newMockCalloutFoo()
	barMock := newMockCalloutBar()

	executor.registerCallout(fooMock)
	executor.registerCallout(barMock)

	// Act
	result := CallSingle(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	require.EqualValues(t, 1, fooMock.fooCount)
	require.EqualValues(t, fooMock.fooTotalCount, result)
	require.Zero(t, barMock.barCount)
}

// Test that only the first callout point is executed if more than one callout
// object was registered.
func TestCallSingleForManyRegisteredCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutBar)(nil)).Elem(),
	})

	mocks := []*mockCalloutFooImpl{
		newMockCalloutFoo(),
		newMockCalloutFoo(),
		newMockCalloutFoo(),
	}

	for _, mock := range mocks {
		executor.registerCallout(mock)
	}

	barMock := newMockCalloutBar()
	fooBarMock := newMockCalloutFooBar()

	executor.registerCallout(fooBarMock)
	executor.registerCallout(barMock)

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

	require.Zero(t, barMock.barCount)
	require.Zero(t, fooBarMock.fooCount)
	require.Zero(t, fooBarMock.barCount)
}

// Test that the default result is returned if no callout object was
// registered.
func TestCallSingleForNoRegisteredCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutBar)(nil)).Elem(),
	})

	// Act
	result := CallSingle(executor, func(callout mockCalloutFoo) int {
		return callout.Foo()
	})

	// Assert
	require.Zero(t, result)
}
