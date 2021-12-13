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
	counter5 := NewBigCounterNaN()
	// Assert
	require.NotNil(t, counter0)
	require.NotNil(t, counter1)
	require.NotNil(t, counter2)
	require.NotNil(t, counter3)
	require.NotNil(t, counter4)
	require.NotNil(t, counter5)
}

// Test addition the big counters in the int64 range.
func TestBigCounterAddInInt64Range(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(37)
	// Act
	sum := counter1.Add(counter2)
	// Assert
	require.EqualValues(t, 42, sum.ToInt64())
	require.EqualValues(t, 5, counter1.ToInt64OrDefault(0, 0))
	require.EqualValues(t, 37, counter2.ToInt64OrDefault(0, 0))
}

// Test addition the NaN and int64 big counters.
func TestBigCounterAddToNaNInt64(t *testing.T) {
	// Arrange
	counterNan := NewBigCounterNaN()
	counterNotNan := NewBigCounter(1)

	// Act
	sumNanNan := counterNan.Add(counterNan)
	sumNanNotNan := counterNan.Add(counterNotNan)
	sumNotNanNan := counterNotNan.Add(counterNan)

	// Assert
	require.EqualValues(t, -1, sumNanNan.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, sumNanNotNan.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, sumNotNanNan.ToInt64OrDefault(-1, -2))
}

// Test addition the NaN and big int big counters.
func TestBigCounterAddToNaNBigIng(t *testing.T) {
	// Arrange
	counterNan := NewBigCounterNaN()
	counterNotNan := NewBigCounter(math.MaxInt64).AddInt64(1)

	// Act
	sumNanNan := counterNan.Add(counterNan)
	sumNanNotNan := counterNan.Add(counterNotNan)
	sumNotNanNan := counterNotNan.Add(counterNan)

	// Assert
	require.EqualValues(t, -1, sumNanNan.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, sumNanNotNan.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, sumNotNanNan.ToInt64OrDefault(-1, -2))
}

// Test addition the int64-based and big int-based big counters.
func TestBigCounterAddInt64AndBigIntRange(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64)
	counter2 := NewBigCounter(1)
	expected := big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
	// Act
	sum1 := counter1.Add(counter2)
	sum2 := counter2.Add(counter1)
	// Assert
	require.EqualValues(t, expected, sum1.ToBigInt())
	require.EqualValues(t, expected, sum2.ToBigInt())
	require.EqualValues(t, math.MaxInt64, counter1.ToInt64OrDefault(0, 0))
	require.EqualValues(t, 1, counter2.ToInt64OrDefault(0, 0))
}

// Test addition the big int-based big counters.
func TestBigCounterAddBigIntRange(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(1)
	expected := big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(math.MaxInt64))
	expected = expected.Add(expected, big.NewInt(2))

	// Act
	sum := counter1.Add(counter2)
	// Assert
	require.EqualValues(t, expected, sum.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), counter2.ToBigInt())
}

// Test additionion multiple big-int big counters.
func TestBigCounterMultipleAddBigInt(t *testing.T) {
	// Arrange
	expected := big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(math.MaxInt64))
	expected = big.NewInt(0).Add(expected, big.NewInt(math.MaxInt64))

	// Act
	counter := NewBigCounter(math.MaxInt64).AddInt64(math.MaxInt64).AddInt64(math.MaxInt64)

	// Assert
	require.EqualValues(t, expected, counter.ToBigInt())
}

// Test addition int64 in place to the int64 counter.
func TestBigCounterAddInt64InPlaceToInt64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(37)

	// Act
	counter1.AddInPlace(counter2)

	// Assert
	require.EqualValues(t, 42, counter1.ToInt64())
	require.EqualValues(t, 37, counter2.ToInt64())
}

// Test addition big int in place to the int64 counter.
func TestBigCounterAddBigIntInPlaceToInt64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(1)

	// Act
	counter1.AddInPlace(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), counter2.ToBigInt())
}

// Test addition int64 in place to the big int counter.
func TestBigCounterAddInt64InPlaceToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter2 := NewBigCounter(5)

	// Act
	counter1.AddInPlace(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(5), counter2.ToBigInt())
}

// Test addition big int in place to the big int counter.
func TestBigCounterAddBigIntInPlaceToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(37)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(5)
	expected := big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(math.MaxInt64))
	expected = expected.Add(expected, big.NewInt(42))

	// Act
	counter1.AddInPlace(counter2)

	// Assert
	require.EqualValues(t, expected, counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(5)), counter2.ToBigInt())
}

// Test addition NaN in place.
func TestBigCounterAddNanInPlace(t *testing.T) {
	// Arrange
	nanCounter1 := NewBigCounterNaN()
	nanCounter2 := NewBigCounterNaN()
	notNanInt64Counter := NewBigCounter(1)
	notNanBigIntCounter := NewBigCounter(math.MaxInt64).AddInt64(1)

	// Act
	nanCounter1.AddInPlace(notNanInt64Counter)
	nanCounter2.AddInPlace(notNanBigIntCounter)
	notNanInt64Counter.AddInPlace(nanCounter1)
	notNanBigIntCounter.AddInPlace(nanCounter1)

	// Assert
	require.EqualValues(t, -1, nanCounter1.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, nanCounter2.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, notNanInt64Counter.ToInt64OrDefault(-1, -2))
	require.EqualValues(t, -1, notNanBigIntCounter.ToInt64OrDefault(-1, -2))
}

// Test add int64 shorthand.
func TestBigCounterAddInt64Shorthand(t *testing.T) {
	// Act
	counter := NewBigCounter(1).AddInt64(5).AddInt64(2).AddInt64(math.MaxInt64).AddInt64(34)
	counterNan := NewBigCounterNaN().AddInt64(5).AddInt64(2).AddInt64(math.MaxInt64).AddInt64(34)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(42)), counter.ToBigInt())
	require.EqualValues(t, -1, counterNan.ToInt64OrDefault(-1, -2))
}

// Test add in place int64 shorthand.
func TestBigCounterAddInPlaceInt64Shorthand(t *testing.T) {
	// Arrange
	counter := NewBigCounter(1)
	counterNan := NewBigCounterNaN()

	// Act
	counter.AddInt64InPlace(5)
	counter.AddInt64InPlace(2)
	counter.AddInt64InPlace(math.MaxInt64)
	counter.AddInt64InPlace(34)
	counterNan.AddInt64InPlace(1)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(42)), counter.ToBigInt())
	require.EqualValues(t, -1, counterNan.ToInt64OrDefault(-1, -2))
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

// Test divide int64 and NaN big counters.
func TestBigCounterDivideNan(t *testing.T) {
	// Arrange
	counterNan := NewBigCounterNaN()
	counterInt64 := NewBigCounter(4)
	counterBigInt := NewBigCounter(math.MaxInt64).AddInt64(1)

	// Act
	nanByNan := counterNan.DivideBy(counterNan)
	nanByInt64 := counterNan.DivideBy(counterInt64)
	nanByBigInt := counterNan.DivideBy(counterBigInt)
	int64ByNan := counterInt64.DivideBy(counterNan)
	bigIntByNan := counterBigInt.DivideBy(counterNan)

	// Assert
	require.True(t, math.IsNaN(nanByNan))
	require.True(t, math.IsNaN(nanByInt64))
	require.True(t, math.IsNaN(nanByBigInt))
	require.True(t, math.IsNaN(int64ByNan))
	require.True(t, math.IsNaN(bigIntByNan))
}

// Test divide big int counters.
func TestBigCounterDivideBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(4)
	counter2 := NewBigCounter(math.MaxInt64).AddInt64(math.MaxInt64).AddInt64(8)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, 0.5, res)
}

// Test divide big int counter by int64 and get result in int64 range.
func TestBigCounterDivideBigIntByInt64InInt64Range(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(math.MaxInt64)
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
	res := counter1.DivideBySafe(counter2)

	// Assert
	require.Zero(t, res)
}

// Test that safe divide works as standard divide.
func TestBigCounterDivideSafe(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxInt64).AddInt64(math.MaxInt64)
	counter2 := NewBigCounter(2)

	// Act
	res := counter1.DivideBySafe(counter2)

	// Assert
	require.EqualValues(t, math.MaxInt64, res)
}

// Test conversion to int64.
func TestBigCounterToInt64(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter4 := NewBigCounterNaN()

	// Act
	value0 := counter0.ToInt64()
	value1 := counter1.ToInt64()
	value2 := counter2.ToInt64()
	value3 := counter3.ToInt64()
	value4 := counter4.ToInt64()

	// Assert
	require.EqualValues(t, 0, value0)
	require.EqualValues(t, -5, value1)
	require.EqualValues(t, math.MaxInt64, value2)
	require.EqualValues(t, math.MaxInt64, value3)
	require.EqualValues(t, 0, value4)
}

// Test conversion to int64 with default above value.
func TestBigCounterToInt64OrDefault(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter4 := NewBigCounterNaN()

	// Act
	value0 := counter0.ToInt64OrDefault(-1, -2)
	value1 := counter1.ToInt64OrDefault(-1, -2)
	value2 := counter2.ToInt64OrDefault(-1, -2)
	value3 := counter3.ToInt64OrDefault(-1, -2)
	value4 := counter4.ToInt64OrDefault(-1, -2)

	// Assert
	require.EqualValues(t, 0, value0)
	require.EqualValues(t, -5, value1)
	require.EqualValues(t, math.MaxInt64, value2)
	require.EqualValues(t, -2, value3)
	require.EqualValues(t, -1, value4)
}

// Test the big counter can be converted to big int.
func TestBigCounterToBigInt(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(-5)
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxInt64).AddInt64(1)
	counter4 := NewBigCounterNaN()

	// Act
	value0 := counter0.ToBigInt()
	value1 := counter1.ToBigInt()
	value2 := counter2.ToBigInt()
	value3 := counter3.ToBigInt()
	value4 := counter4.ToBigInt()

	// Assert
	require.EqualValues(t, big.NewInt(0), value0)
	require.EqualValues(t, big.NewInt(-5), value1)
	require.EqualValues(t, big.NewInt(math.MaxInt64), value2)
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), value3)
	require.EqualValues(t, big.NewInt(0), value4)
}
