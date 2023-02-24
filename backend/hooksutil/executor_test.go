package hooksutil

import (
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Foo callout specification.
type mockCalloutSpecificationFoo interface {
	Foo() int
}

// Bar callout specification.
type mockCalloutSpecificationBar interface {
	Bar() bool
}

// FooBar callout specification.
type mockCalloutSpecificationFooBar interface {
	mockCalloutSpecificationFoo
	mockCalloutSpecificationBar
}

// Foo mock callout carrier implementation.
type mockCalloutCarrierFooImpl struct {
	fooCount             int
	fooLastCallTimestamp int64
	closeCount           int
	closeErr             error
}

// Constructs an instance of the mock callout carrier.
func newMockCalloutCarrierFoo() *mockCalloutCarrierFooImpl {
	return &mockCalloutCarrierFooImpl{
		fooCount:   0,
		closeCount: 0,
		closeErr:   nil,
	}
}

// Counts the call count.
func (c *mockCalloutCarrierFooImpl) Foo() int {
	c.fooCount++
	c.fooLastCallTimestamp = time.Now().UnixNano()
	return c.fooCount
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
		reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem(),
	})

	// Assert
	require.NotNil(t, emptyExecutor)
	require.NotNil(t, nilExecutor)
	require.NotNil(t, executor)

	require.Contains(t, executor.registeredCarriers, reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem())
}

// Test that the hook executor constructor panics on an invalid type (it's a
// programming bug).
func TestNewHookExecutorInvalidType(t *testing.T) {
	// Arrange
	// Missing .Elem() call
	invalidType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil))

	// Assert
	require.Panics(t, func() {
		// Act
		_ = NewHookExecutor([]reflect.Type{invalidType})
	})
}

// Test that the supported callout carrier is registered properly.
func TestRegisterSupportedCalloutCarrier(t *testing.T) {
	// Arrange
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		specificationType,
	})

	// Act
	executor.registerCalloutCarrier(newMockCalloutCarrierFoo())

	// Assert
	require.NotEmpty(t, executor.registeredCarriers[specificationType])
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
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		specificationType,
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

	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		specificationType,
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
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		specificationType,
	})

	executor.registerCalloutCarrier(newMockCalloutCarrierFoo())

	// Act
	isRegistered := executor.HasRegistered(specificationType)

	// Assert
	require.True(t, isRegistered)
}

// Test that the non-registered callout carrier is non detected as registered.
func TestHasRegisteredForNonRegisteredCalloutCarrier(t *testing.T) {
	// Arrange
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{
		specificationType,
	})

	// Act
	isRegistered := executor.HasRegistered(specificationType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the unsupported specification is not detected as registered.
func TestHasRegisteredForUnsupportedSpecification(t *testing.T) {
	// Arrange
	specificationType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	isRegistered := executor.HasRegistered(specificationType)

	// Assert
	require.False(t, isRegistered)
}

// Test that the types of the supported callout specifications are returned properly.
func TestGetTypesOfSupportedCarrierSpecifications(t *testing.T) {
	// Arrange
	fooType := reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem()
	barType := reflect.TypeOf((*mockCalloutSpecificationBar)(nil)).Elem()

	executor := NewHookExecutor([]reflect.Type{
		fooType,
		barType,
	})

	// Act
	supportedTypes := executor.GetTypesOfSupportedCalloutSpecifications()

	// Assert
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, fooType)
	require.Contains(t, supportedTypes, barType)
}

// Test that the callouts are called in the sequential order properly.
func TestCallSequential(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem(),
		reflect.TypeOf((*mockCalloutSpecificationBar)(nil)).Elem(),
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
	results := CallSequential(executor, func(carrier mockCalloutSpecificationFoo) int {
		return carrier.Foo()
	})

	// Assert
	// 1. One result for each callout object.
	require.Len(t, results, len(fooMocks)+1)

	for i, mock := range fooMocks {
		result := results[i]

		// 2. Has expected output.
		require.EqualValues(t, 1, result)
		// 3. The callout was called exactly once.
		require.EqualValues(t, 1, mock.fooCount)

		if i != 0 {
			previousMock := fooMocks[i-1]
			// 4. The callouts were executed in an expected order.
			require.LessOrEqual(t, previousMock.fooLastCallTimestamp, mock.fooLastCallTimestamp)
		}
	}

	// 5. FooBar mock should be called too.
	require.EqualValues(t, 1, fooBarMock.fooCount)
	require.Zero(t, fooBarMock.barCount)

	// 6. Bar mock shouldn't be called.
	require.Zero(t, barMock.barCount)
}

// Test that the empty output and false status is returned if there is no
// carriers registered.
func TestCallSequentialUntilProcessedForMissingCarriers(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{})

	// Act
	ok, result := CallSequentialUntilProcessed(executor, func(carrier mockCalloutSpecificationFoo) (CallStatus, int) {
		return CallStatusProcessed, carrier.Foo()
	})

	// Assert
	require.False(t, ok)
	require.Zero(t, result)
}

// Test that the carrier execution stops on first callout that processed the
// data.
func TestCallSequentialUntilProcessed(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutSpecificationFooBar)(nil)).Elem(),
	})

	mocks := []*mockCalloutCarrierFooBarImpl{
		newMockCalloutCarrierFooBar(),
		newMockCalloutCarrierFooBar(),
		newMockCalloutCarrierFooBar(),
	}

	for _, mock := range mocks {
		executor.registerCalloutCarrier(mock)
	}

	carrierIdx := -1

	// Act
	ok, result := CallSequentialUntilProcessed(executor, func(carrier mockCalloutSpecificationFooBar) (CallStatus, int) {
		carrierIdx++
		carrier.Bar()
		if carrierIdx == 1 {
			return CallStatusProcessed, carrier.Foo()
		}
		return CallStatusSkipped, 0
	})

	// Assert
	require.True(t, ok)
	require.EqualValues(t, 1, result)

	require.Zero(t, 0, mocks[0].fooCount)
	require.EqualValues(t, 1, mocks[0].barCount)

	require.EqualValues(t, 1, mocks[1].fooCount)
	require.EqualValues(t, 1, mocks[1].barCount)

	require.Zero(t, mocks[2].fooCount)
	require.Zero(t, mocks[2].barCount)
}

// Test that the empty output and false status is returned if all registered
// carrier return the skipped status.
func TestCallSequentialUntilProcessedAllSkipped(t *testing.T) {
	// Arrange
	executor := NewHookExecutor([]reflect.Type{
		reflect.TypeOf((*mockCalloutSpecificationFoo)(nil)).Elem(),
	})

	mocks := []*mockCalloutCarrierFooImpl{
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
		newMockCalloutCarrierFoo(),
	}

	for _, mock := range mocks {
		executor.registerCalloutCarrier(mock)
	}

	// Act
	ok, result := CallSequentialUntilProcessed(executor, func(carrier mockCalloutSpecificationFoo) (CallStatus, int) {
		return CallStatusSkipped, 42
	})

	// Assert
	require.False(t, ok)
	require.Zero(t, result)
}
