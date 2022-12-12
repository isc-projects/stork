package hooksutil

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Mock callout carrier interfaces.
type mockCalloutCarrierFoo interface {
	Foo() int
	Close() error
}

type mockCalloutCarrierBar interface {
	Bar() bool
	Close() error
}

// Foo mock callout carrier implementation.
type mockCalloutCarrierFooImpl struct {
	fooCount int
	// The call count including calls from all mock instances.
	fooTotalCount int
	closeCount    int
	closeErr      error
}

// The Foo call counter shared between all mock instances.
var sharedFooCount int //nolint:gochecknoglobals

// Constructs an instance of the mock callout carrier.
func newMockCalloutCarrierFoo() *mockCalloutCarrierFooImpl {
	return &mockCalloutCarrierFooImpl{
		fooCount:      0,
		fooTotalCount: 0,
		closeCount:    0,
		closeErr:      nil,
	}
}

// Counts the call count.
func (c *mockCalloutCarrierFooImpl) Foo() int {
	c.fooCount++
	sharedFooCount++
	c.fooTotalCount = sharedFooCount
	return c.fooTotalCount
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutCarrierFooImpl) Close() error {
	c.closeCount++
	return c.closeErr
}

// Bar mock implementation.
type mockCalloutCarrierBarImpl struct {
	barCount   int
	closeCount int
	closeErr   error
}

// Constructs the Bar mock.
func newMockCalloutCarrierBar() *mockCalloutCarrierBarImpl {
	return &mockCalloutCarrierBarImpl{}
}

// Counts the calls. Return parity of an actual value.
func (c *mockCalloutCarrierBarImpl) Bar() bool {
	c.barCount++
	return c.barCount%2 == 0
}

// It counts the call count and returns the mocked error.
func (c *mockCalloutCarrierBarImpl) Close() error {
	c.closeCount++
	return c.closeErr
}

// FooBar mock implementation.
type mockCalloutCarrierFooBarImpl struct {
	mockCalloutCarrierFooImpl
	mockCalloutCarrierBarImpl
}

// Constructs the FooBar mock.
func newMockCalloutCarrierFooBar() *mockCalloutCarrierFooBarImpl {
	return &mockCalloutCarrierFooBarImpl{}
}

func (c *mockCalloutCarrierFooBarImpl) Close() error {
	return c.mockCalloutCarrierFooImpl.Close()
}

// Test that the hook executor is constructed properly.
func TestNewHookExecutor(t *testing.T) {
	// Arrange & Act
	emptyExecutor := NewHookExecutor([]reflect.Type{})
	nilExecutor := NewHookExecutor(nil)
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem(),
	})

	// Assert
	require.NotNil(t, emptyExecutor)
	require.NotNil(t, nilExecutor)
	require.NotNil(t, executor)

	require.Contains(t, executor.registeredCarriers, reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem())
}

// Test that the hook executor constructor panics on an invalid type (it's a
// programming bug).
func TestNewHookExecutorInvalidType(t *testing.T) {
	// Arrange
	// Missing .Elem() call
	invalidType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil))

	// Assert
	require.Panics(t, func() {
		// Act
		_ = NewHookExecutor([]reflect.Type{invalidType})
	})
}

// Test that the supported callout carrier is registered properly.
func TestRegisterSupportedCalloutCarrier(t *testing.T) {
	// Arrange
	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		carrierType,
	})

	// Act
	executor.registerCalloutCarrier(newMockCalloutCarrierFoo())

	// Assert
	require.NotEmpty(t, executor.registeredCarriers[carrierType])
}

// Test that the unsupported callout carrier is not registered.
func TestRegisterUnsupportedCalloutCarrier(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	executor.registerCalloutCarrier(newMockCalloutCarrierFoo())

	// Assert
	require.Empty(t, executor.registeredCarriers)
}

// Test that all callout carriers are unregistering.
func TestUnregisterAllCalloutCarriers(t *testing.T) {
	// Arrange
	carrier := newMockCalloutCarrierFoo()
	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		carrierType,
	})
	executor.registerCalloutCarrier(carrier)

	// Act
	errs := executor.unregisterAllCalloutCarriers()

	// Assert
	require.Empty(t, executor.registeredCarriers)
	require.EqualValues(t, 1, carrier.closeCount)
	require.Empty(t, errs)
}

// Test that if one callout carrier returns an error, other are unregistered
// properly.
func TestUnregisterAllCalloutCarriersWithError(t *testing.T) {
	// Arrange
	successCarrier := newMockCalloutCarrierFoo()
	failedCarrier := newMockCalloutCarrierFoo()
	failedCarrier.closeErr = errors.New("Close failed")

	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		carrierType,
	})

	executor.registerCalloutCarrier(successCarrier)
	executor.registerCalloutCarrier(failedCarrier)

	// Act
	errs := executor.unregisterAllCalloutCarriers()

	// Assert
	require.Len(t, errs, 1)
	require.EqualValues(t, 1, successCarrier.closeCount)
	require.EqualValues(t, 1, failedCarrier.closeCount)
}

// Test that the registered callout carrier is detected as registered.
func TestHasRegisteredForRegisteredCalloutCarrier(t *testing.T) {
	// Arrange
	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		carrierType,
	})

	executor.registerCalloutCarrier(newMockCalloutCarrierFoo())

	// Act
	isRegistered := executor.HasRegistered(carrierType)

	// Assert
	require.True(t, isRegistered)
}

// Test that the non-registered callout carrier is non detected as registered.
func TestHasRegisteredForNonRegisteredCalloutCarrier(t *testing.T) {
	// Arrange
	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		carrierType,
	})

	// Act
	isRegistered := executor.HasRegistered(carrierType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the unsupported callout carrier is not detected as registered.
func TestHasRegisteredForUnsupportedCalloutCarrier(t *testing.T) {
	// Arrange
	carrierType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	isRegistered := executor.HasRegistered(carrierType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the supported callout types are returned properly.
func TestGetSupportedCarrierTypes(t *testing.T) {
	// Arrange
	fooType := reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem()
	barType := reflect.TypeOf((*mockCalloutCarrierBar)(nil)).Elem()

	executor := NewHookExecutor([]reflect.Type{
		fooType,
		barType,
	})

	// Act
	supportedTypes := executor.GetSupportedCalloutCarrierTypes()

	// Assert
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, fooType)
	require.Contains(t, supportedTypes, barType)
}

// Test that the callouts are called in the sequential order properly.
func TestCallSequential(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutCarrierBar)(nil)).Elem(),
	})

	fooMocks := []*mockCalloutCarrierFooImpl{
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
	}
	barMock := newMockCalloutCarrierBar()
	fooBarMock := newMockCalloutCarrierFooBar()

	for _, mock := range fooMocks {
		executor.registerCalloutCarrier(mock)
	}
	executor.registerCalloutCarrier(barMock)
	executor.registerCalloutCarrier(fooBarMock)

	// Act
	results := CallSequential(executor, func(callout mockCalloutCarrierFoo) int {
		return callout.Foo()
	})

	// Assert
	// 1. One result for each callout object.
	require.Len(t, results, len(fooMocks)+1)

	for i, mock := range fooMocks {
		result := results[i]

		// 2. Has expected output.
		require.EqualValues(t, mock.fooTotalCount, result)
		// 3. The callout was called exactly once.
		require.EqualValues(t, 1, mock.fooCount)

		if i != 0 {
			previousMock := fooMocks[i-1]
			// 4. The callouts were executed in an expected order.
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

// Test that the callout is executed properly if exactly one callout carrier
// was registered.
func TestCallSingleForOneRegisteredCallout(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutCarrierBar)(nil)).Elem(),
	})

	fooMock := newMockCalloutCarrierFoo()
	barMock := newMockCalloutCarrierBar()

	executor.registerCalloutCarrier(fooMock)
	executor.registerCalloutCarrier(barMock)

	// Act
	result := CallSingle(executor, func(carrier mockCalloutCarrierFoo) int {
		return carrier.Foo()
	})

	// Assert
	require.EqualValues(t, 1, fooMock.fooCount)
	require.EqualValues(t, fooMock.fooTotalCount, result)
	require.Zero(t, barMock.barCount)
}

// Test that only the first callout is executed if more than one carrier was
// registered.
func TestCallSingleForManyRegisteredCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutCarrierBar)(nil)).Elem(),
	})

	mocks := []*mockCalloutCarrierFooImpl{
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
	}

	for _, mock := range mocks {
		executor.registerCalloutCarrier(mock)
	}

	barMock := newMockCalloutCarrierBar()
	fooBarMock := newMockCalloutCarrierFooBar()

	executor.registerCalloutCarrier(fooBarMock)
	executor.registerCalloutCarrier(barMock)

	// Act
	result := CallSingle(executor, func(carrier mockCalloutCarrierFoo) int {
		return carrier.Foo()
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

// Test that the default result is returned if no callout carrier was
// registered.
func TestCallSingleForNoRegisteredCallouts(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutCarrierFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutCarrierBar)(nil)).Elem(),
	})

	// Act
	result := CallSingle(executor, func(carrier mockCalloutCarrierFoo) int {
		return carrier.Foo()
	})

	// Assert
	require.Zero(t, result)
}
