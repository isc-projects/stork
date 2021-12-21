package storkutil

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the big counter is properly constructed.
func TestBigCounterConstruct(t *testing.T) {
	// Act
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(42)
	counter2 := NewBigCounter(-1)
	counter3 := NewBigCounter(math.MaxInt64)
	counter4 := NewBigCounter(-math.MaxInt64)
	// Assert
	require.NotNil(t, counter0)
	require.NotNil(t, counter1)
	require.NotNil(t, counter2)
	require.NotNil(t, counter3)
	require.NotNil(t, counter4)
}

// Test addition int64 in place to the int64 counter.
func TestBigCounterAddInt64ToInt64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(37)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, 42, counter1.ToInt64())
	require.EqualValues(t, 37, counter2.ToInt64())
}

// Test addition big int in place to the int64 counter.
func TestBigCounterAddBigIntToInt64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(1)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), counter2.ToBigInt())
}

// Test addition int64 in place to the big int counter.
func TestBigCounterAddInt64ToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter2 := NewBigCounter(5)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(5), counter2.ToBigInt())
}

// Test addition big int in place to the big int counter.
func TestBigCounterAddBigIntToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(37)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(5)
	expected := big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(math.MaxInt64))
	expected = expected.Add(expected, big.NewInt(42))

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, expected, counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(5)), counter2.ToBigInt())
}

// Test add in place int64 shorthand.
func TestBigCounterAddInt64Shorthand(t *testing.T) {
	// Arrange
	counter := NewBigCounter(1)

	// Act
	counter.AddInt64(5)
	counter.AddInt64(2)
	counter.AddInt64(math.MaxInt64)
	counter.AddInt64(34)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(42)), counter.ToBigInt())
}

// Test add in place uint64 shorthand.
func TestBigCounterAddUInt64Shorthand(t *testing.T) {
	// Arrange
	expected := big.NewInt(0).Add(
		big.NewInt(0).Add(
			big.NewInt(0).SetUint64(math.MaxUint64),
			big.NewInt(0).SetUint64(math.MaxUint64),
		),
		big.NewInt(42),
	)

	// Act
	counter1 := NewBigCounter(1)
	counter1.AddUInt64(uint64(41))
	counter1.AddUInt64(math.MaxUint64)
	counter1.AddUInt64(math.MaxUint64)
	var val int64 = -1
	counter2 := NewBigCounter(0).AddUInt64(uint64(val))

	// Assert
	require.EqualValues(t,
		expected,
		counter1.ToBigInt())

	require.EqualValues(t, big.NewInt(0).SetUint64(math.MaxUint64), counter2.ToBigInt())
}

// Test divide int64 big counters.
func TestBigCounterDivideInt64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(2)
	counter2 := NewBigCounter(4)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, 0.5, res)
}

// Test divide big int counters.
func TestBigCounterDivideBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddUInt64(4)
	counter2 := NewBigCounter(math.MaxInt64).AddUInt64(math.MaxInt64).AddUInt64(8)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, 0.5, res)
}

// Test divide big int counter by int64 and get result in int64 range.
func TestBigCounterDivideBigIntByInt64InInt64Range(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddUInt64(math.MaxInt64)
	counter2 := NewBigCounter(2)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, math.MaxInt64, res)
}

// Test that safe divide doesn't panic.
func TestBigCounterSafeDivideByZero(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(1)
	counter2 := NewBigCounter(0)

	// Act
	res := counter1.DivideSafeBy(counter2)

	// Assert
	require.Zero(t, res)
}

// Test that safe divide works as standard divide.
func TestBigCounterDivideSafe(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddUInt64(math.MaxInt64)
	counter2 := NewBigCounter(2)

	// Act
	res := counter1.DivideSafeBy(counter2)

	// Assert
	require.EqualValues(t, math.MaxInt64, res)
}

// Test conversion to int64.
func TestBigCounterToInt64(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxInt64).AddUInt64(1)

	// Act
	value0 := counter0.ToInt64()
	value1 := counter1.ToInt64()
	value2 := counter2.ToInt64()
	value3 := counter3.ToInt64()

	// Assert
	require.EqualValues(t, 0, value0)
	require.EqualValues(t, -5, value1)
	require.EqualValues(t, math.MaxInt64, value2)
	require.EqualValues(t, math.MaxInt64, value3)
}

// Test conversion to uint64.
func TestBigCounterToUint64(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(0).AddUInt64(math.MaxUint64)
	counter3 := NewBigCounter(math.MaxInt64).AddUInt64(1)

	// Act
	value0 := counter0.ToUint64()
	value1 := counter1.ToUint64()
	value2 := counter2.ToUint64()
	value3 := counter3.ToUint64()

	// Assert
	require.EqualValues(t, 0, value0)
	require.EqualValues(t, 0, value1)
	require.EqualValues(t, uint64(math.MaxUint64), value2)
	require.EqualValues(t, uint64(math.MaxInt64)+1, value3)
}

// Test the big counter can be converted to big int.
func TestBigCounterToBigInt(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxInt64).AddUInt64(1)

	// Act
	value0 := counter0.ToBigInt()
	value1 := counter1.ToBigInt()
	value2 := counter2.ToBigInt()
	value3 := counter3.ToBigInt()

	// Assert
	require.EqualValues(t, big.NewInt(0), value0)
	require.EqualValues(t, big.NewInt(-5), value1)
	require.EqualValues(t, big.NewInt(math.MaxInt64), value2)
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), value3)
}

// Benchmarks.
// The below benchmark measure the big counter performance.
// The big counter is 4.8 times faster then raw big int,
// but 78 times slower then raw int64.

// Common maxInt64 denominator for all benchmarks.
// It is used to increase the operations in the benchmark function.
const denominator int64 = 1000000

// Benchmark the addition to the big counter in int64 range.
func BenchmarkBigCounterInt64InInt64Range(b *testing.B) {
	// Arrange
	counter := NewBigCounter(0)

	// Act
	var factor int64 = math.MaxInt64 / denominator
	var cumulativeSum int64 = 0
	for cumulativeSum < math.MaxInt64-factor {
		counter.AddInt64(factor)
		cumulativeSum += factor
	}
}

// Benchmark the addition to the big int in int64 range.
func BenchmarkBigCounterBigIntInInt64Range(b *testing.B) {
	// Arrange
	counter := big.NewInt(0)

	// Act
	var factor int64 = math.MaxInt64 / denominator
	var cumulativeSum int64 = 0
	for cumulativeSum < math.MaxInt64-factor {
		counter.Add(counter, big.NewInt(factor))
		cumulativeSum += factor
	}
}

// Benchmark the addition to the int64 in int64 range.
func BenchmarkBigCounterStandardInt64InInt64Range(b *testing.B) {
	// Arrange
	var counter int64 = 0

	// Act
	var factor int64 = math.MaxInt64 / denominator
	var cumulativeSum int64 = 0
	for cumulativeSum < math.MaxInt64-factor {
		counter += factor
		cumulativeSum += factor
	}
}
