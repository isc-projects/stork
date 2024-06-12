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
	counter2 := NewBigCounter(math.MaxInt64)
	counter3 := NewBigCounter(math.MaxUint64)
	// Assert
	require.NotNil(t, counter0)
	require.NotNil(t, counter1)
	require.NotNil(t, counter2)
	require.NotNil(t, counter3)
}

// Test addition uint64 in place to the uint64 counter.
func TestBigCounterAddUint64ToUint64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(37)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, 42, counter1.ToInt64())
	require.EqualValues(t, 37, counter2.ToInt64())
}

// Test addition big int in place to the uint64 counter.
func TestBigCounterAddBigIntToUint64(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(5)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(1)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(1)), counter2.ToBigInt())
}

// Test addition uint64 in place to the big int counter.
func TestBigCounterAddUint64ToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxUint64).AddUint64(1)
	counter2 := NewBigCounter(5)

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(6)), counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(5), counter2.ToBigInt())
}

// Test addition big int in place to the big int counter.
func TestBigCounterAddBigIntToBigInt(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxUint64).AddUint64(37)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(5)
	expected := big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(0).SetUint64(math.MaxUint64))
	expected = expected.Add(expected, big.NewInt(42))

	// Act
	counter1.Add(counter2)

	// Assert
	require.EqualValues(t, expected, counter1.ToBigInt())
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(5)), counter2.ToBigInt())
}

// Test add in place uint64 shorthand.
func TestBigCounterAddUint64Shorthand(t *testing.T) {
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
	counter1.AddUint64(uint64(41))
	counter1.AddUint64(math.MaxUint64)
	counter1.AddUint64(math.MaxUint64)
	var val int64 = -1
	counter2 := NewBigCounter(0).AddUint64(uint64(val))

	// Assert
	require.EqualValues(t,
		expected,
		counter1.ToBigInt())

	require.EqualValues(t, big.NewInt(0).SetUint64(math.MaxUint64), counter2.ToBigInt())
}

// Test add in place big.Int shorthand.
func TestBigCounterAddBigIntShorthand(t *testing.T) {
	// Arrange
	expected := big.NewInt(0).Add(
		big.NewInt(0).Add(
			big.NewInt(111),
			big.NewInt(0).SetUint64(math.MaxUint64),
		),
		big.NewInt(0).SetUint64(math.MaxUint64),
	)
	// Act
	counter := NewBigCounter(1)
	_, ok1 := counter.AddBigInt(big.NewInt(10))
	_, ok2 := counter.AddBigInt(big.NewInt(100))
	_, ok3 := counter.AddBigInt(
		big.NewInt(0).Add(
			big.NewInt(0).SetUint64(math.MaxUint64),
			big.NewInt(0).SetUint64(math.MaxUint64),
		),
	)
	// Assert
	require.True(t, ok1)
	require.True(t, ok2)
	require.True(t, ok3)
	require.EqualValues(t, expected, counter.ToBigInt())
}

// Test that add in place big.Int ignores the negative numbers.
func TestBigCounterAddBigIntShorthandIgnoreNegatives(t *testing.T) {
	// Arrange & Act
	counter := NewBigCounter(42)
	_, ok1 := counter.AddBigInt(big.NewInt(-1))
	_, ok2 := counter.AddBigInt(big.NewInt(-2))
	_, ok3 := counter.AddBigInt(big.NewInt(math.MinInt64))
	_, ok4 := counter.AddBigInt(big.NewInt(0).Add(
		big.NewInt(math.MinInt64),
		big.NewInt(math.MinInt64),
	))
	// Assert
	require.False(t, ok1)
	require.False(t, ok2)
	require.False(t, ok3)
	require.False(t, ok4)
	require.EqualValues(t, big.NewInt(42), counter.ToBigInt())
}

// Test divide uint64 big counters.
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
	counter1 := NewBigCounter(math.MaxUint64).AddUint64(4)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(math.MaxUint64).AddUint64(8)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, 0.5, res)
}

// Test divide big int counter by uint64 and get result in uint64 range.
func TestBigCounterDivideBigIntByInt64InInt64Range(t *testing.T) {
	// Arrange
	counter1 := NewBigCounter(math.MaxUint64).AddUint64(math.MaxUint64)
	counter2 := NewBigCounter(2)

	// Act
	res := counter1.DivideBy(counter2)

	// Assert
	require.EqualValues(t, float64(math.MaxUint64), res)
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
	counter1 := NewBigCounter(math.MaxUint64).AddUint64(math.MaxUint64)
	counter2 := NewBigCounter(2)

	// Act
	res := counter1.DivideSafeBy(counter2)

	// Assert
	require.EqualValues(t, float64(math.MaxUint64), res)
}

// Test conversion to int64.
func TestBigCounterToInt64(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(math.MaxUint64)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(1)

	// Act
	value0 := counter0.ToInt64()
	value1 := counter1.ToInt64()
	value2 := counter2.ToInt64()

	// Assert
	require.EqualValues(t, 0, value0)
	require.EqualValues(t, math.MaxInt64, value1)
	require.EqualValues(t, math.MaxInt64, value2)
}

// Test conversion to uint64.
func TestBigCounterToUint64(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(0).AddUint64(math.MaxUint64)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(1)

	// Act
	value0, ok0 := counter0.ToUint64()
	value1, ok1 := counter1.ToUint64()
	value2, ok2 := counter2.ToUint64()

	// Assert
	require.EqualValues(t, 0, value0)
	require.True(t, ok0)
	require.EqualValues(t, uint64(math.MaxUint64), value1)
	require.True(t, ok1)
	require.EqualValues(t, uint64(math.MaxUint64), value2)
	require.False(t, ok2)
}

// Test conversion to float64.
func TestBigCounterToFloat64(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		// Arrange & Act
		counter := NewBigCounter(0).ToFloat64()

		// Assert
		require.Zero(t, counter)
	})

	t.Run("small int", func(t *testing.T) {
		// Arrange & Act
		counter := NewBigCounter(42).ToFloat64()

		// Assert
		require.EqualValues(t, 42, uint64(counter))
	})

	t.Run("max safe integer", func(t *testing.T) {
		// Arrange
		value := (int64(1) << 53) - 1
		counter := NewBigCounterFromInt64(value)

		// Act & Assert
		require.EqualValues(t, value, int64(counter.ToFloat64()))
	})

	t.Run("above max safe integer", func(t *testing.T) {
		// Arrange
		value := (int64(1) << 53) + 1
		counter := NewBigCounterFromInt64(value)

		// Act & Assert
		// Warning! This test may be unstable because it depends on the
		// implementation details of floating point numbers.
		require.NotEqualValues(t, value, int64(counter.ToFloat64()))
		require.EqualValues(t, value-1, int64(counter.ToFloat64()))
	})

	t.Run("big int", func(t *testing.T) {
	})
}

// Test the big counter can be converted to big int.
func TestBigCounterToBigInt(t *testing.T) {
	// Arrange
	counter0 := NewBigCounter(0)
	counter1 := NewBigCounter(math.MaxUint64)
	counter2 := NewBigCounter(math.MaxUint64).AddUint64(1)

	// Act
	value0 := counter0.ToBigInt()
	value1 := counter1.ToBigInt()
	value2 := counter2.ToBigInt()

	// Assert
	require.EqualValues(t, big.NewInt(0), value0)
	require.EqualValues(t, big.NewInt(0).SetUint64(math.MaxUint64), value1)
	require.EqualValues(t, big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(1)), value2)
}

// Test the big counter can be converted to native type.
func TestBigCounterToNativeType(t *testing.T) {
	// Arrange
	counterBase := NewBigCounter(42)
	counterExtended := NewBigCounter(math.MaxUint64).AddUint64(1)

	// Act
	nativeBase := counterBase.ConvertToNativeType()
	nativeExtended := counterExtended.ConvertToNativeType()

	// Assert
	require.EqualValues(t, uint64(42), nativeBase)
	require.EqualValues(t, big.NewInt(0).Add(
		big.NewInt(0).SetUint64(math.MaxUint64),
		big.NewInt(1),
	), nativeExtended)
}

// Test the big counter can be constructed from the int64.
func TestBigCounterConstructFromInt64(t *testing.T) {
	t.Run("Construct from zero", func(t *testing.T) {
		// Arrange
		val := int64(0)

		// Act
		counter := NewBigCounterFromInt64(val)

		// Assert
		require.NotNil(t, counter)
		require.EqualValues(t, big.NewInt(0), counter.ToBigInt())
	})

	t.Run("Construct from positive value", func(t *testing.T) {
		// Arrange
		val := int64(42)

		// Act
		counter := NewBigCounterFromInt64(val)

		// Assert
		require.NotNil(t, counter)
		require.EqualValues(t, big.NewInt(42), counter.ToBigInt())
	})

	t.Run("Construct from negative value", func(t *testing.T) {
		// Arrange
		val := int64(-1)

		// Act
		counter := NewBigCounterFromInt64(val)

		// Assert
		require.Nil(t, counter)
	})
}

// Test the big counter can be constructed from the big int.
func TestBigCounterConstructFromBigInt(t *testing.T) {
	t.Run("Construct from zero", func(t *testing.T) {
		// Arrange
		bigInt := big.NewInt(0)

		// Act
		counter := NewBigCounterFromBigInt(bigInt)

		// Assert
		require.NotNil(t, counter)
		require.EqualValues(t, bigInt, counter.ToBigInt())
	})

	t.Run("Construct from uint64 range", func(t *testing.T) {
		// Arrange
		bigInt := big.NewInt(0).SetUint64(math.MaxUint64)

		// Act
		counter := NewBigCounterFromBigInt(bigInt)

		// Assert
		require.NotNil(t, counter)
		require.EqualValues(t, bigInt, counter.ToBigInt())
	})

	t.Run("Construct from above uint64 range", func(t *testing.T) {
		// Arrange
		bigInt := big.NewInt(0).Add(
			big.NewInt(0).SetUint64(math.MaxUint64),
			big.NewInt(1),
		)

		// Act
		counter := NewBigCounterFromBigInt(bigInt)

		// Assert
		require.NotNil(t, counter)
		require.EqualValues(t, bigInt, counter.ToBigInt())
	})

	t.Run("Construct from negative value", func(t *testing.T) {
		// Arrange
		bigInt := big.NewInt(-1)

		// Act
		counter := NewBigCounterFromBigInt(bigInt)

		// Assert
		require.Nil(t, counter)
	})
}

// Benchmarks.
// The below benchmark measure the big counter performance.

// Benchmark the addition to the big counter in uint64 range.
func BenchmarkBigCounterInUint64Range(b *testing.B) {
	counter := NewBigCounter(0)

	for i := 0; i < b.N; i++ {
		counter.AddUint64(1)
	}
}

// Benchmark the addition to the big counter out of uint64 range.
func BenchmarkBigCounterOutUint64Range(b *testing.B) {
	counter := NewBigCounter(math.MaxUint64)
	for i := 0; i < b.N; i++ {
		counter.AddUint64(1)
	}
}

// Benchmark the addition to the big int in uint64 range.
func BenchmarkBigIntInUint64Range(b *testing.B) {
	counter := big.NewInt(0)

	for i := 0; i < b.N; i++ {
		counter.Add(counter, big.NewInt(1))
	}
}

// Benchmark the addition to the big int out of uint64 range.
func BenchmarkBigIntOutUint64Range(b *testing.B) {
	counter := big.NewInt(0).SetUint64(math.MaxUint64)

	for i := 0; i < b.N; i++ {
		counter.Add(counter, big.NewInt(1))
	}
}

// Benchmark the addition to the uint64 in uint64 range.
func BenchmarkStandardUint64InUint64Range(b *testing.B) {
	counter := uint64(0)

	for i := 0; i < b.N; i++ {
		counter++
	}
}
