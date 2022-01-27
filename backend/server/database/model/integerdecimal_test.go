package dbmodel

import (
	"math/big"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Test construct the integer decimal.
func TestConstructIntegerDecimal(t *testing.T) {
	// Act
	integerDecimal := newIntegerDecimal(big.NewInt(42))
	nilIntegerDecimal := newIntegerDecimal(nil)
	zeroIntegerDecimal := newIntegerDecimalZero()

	// Assert
	require.EqualValues(t, big.NewInt(42), &integerDecimal.Int)
	require.Nil(t, nilIntegerDecimal)
	require.EqualValues(t, big.NewInt(0), &zeroIntegerDecimal.Int)
}

// Test that the zero big interger is serialized to bytes.
func TestAppendValueBigIntZero(t *testing.T) {
	// Arrange
	integerDecimal := newIntegerDecimalZero()

	// Act
	bytes, err := integerDecimal.AppendValue([]byte{}, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte("0"), bytes)
}

// Test that the zero big interger is serialized to bytes with quotes.
func TestAppendValueBigIntZeroWithQuotes(t *testing.T) {
	// Arrange
	integerDecimal := newIntegerDecimalZero()

	// Act
	bytes, err := integerDecimal.AppendValue([]byte{}, 1)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte(`'0'`), bytes)
}

// Test that the very big interger is serialized to bytes.
func TestAppendValueVeryBigInt(t *testing.T) {
	// Arrange
	str := "1234567801234567801234567890123456789012345678901234567801234567890"
	bigInt, _ := big.NewInt(0).SetString(str, 10)
	integerDecimal := newIntegerDecimal(bigInt)

	// Act
	bytes, err := integerDecimal.AppendValue([]byte{}, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte(str), bytes)
}

// Test that the negative big interger is serialized to bytes.
func TestAppendValueNegativeBigInt(t *testing.T) {
	// Arrange
	integerDecimal := newIntegerDecimal(big.NewInt(-1))

	// Act
	bytes, err := integerDecimal.AppendValue([]byte{}, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []byte("-1"), bytes)
}

// Go-PG reader mock.
type poolReaderMock struct {
	bytes []byte
	err   error
}

func (r *poolReaderMock) Buffered() int {
	panic("not implemented")
}

func (r *poolReaderMock) Bytes() []byte {
	panic("not implemented")
}

func (r *poolReaderMock) Read([]byte) (int, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadByte() (byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) UnreadByte() error {
	panic("not implemented")
}

func (r *poolReaderMock) ReadSlice(byte) ([]byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) Discard(int) (int, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadFull() ([]byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadFullTemp() ([]byte, error) {
	return r.bytes, r.err
}

// Test that the zero big integers is deserialized from bytes.
func TestScanValueZeroBigInt(t *testing.T) {
	// Arrange
	reader := &poolReaderMock{[]byte("1"), nil}
	integerDecimal := newIntegerDecimalZero()

	// Act
	err := integerDecimal.ScanValue(reader, 1)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(1), &integerDecimal.Int)
}

// Test that the very big integers is deserialized from bytes.
func TestScanValueVeryBigInt(t *testing.T) {
	// Arrange
	str := "1234567801234567801234567890123456789012345678901234567801234567890"
	reader := &poolReaderMock{[]byte(str), nil}
	integerDecimal := newIntegerDecimalZero()
	expectedBigInt, _ := big.NewInt(0).SetString(str, 10)

	// Act
	err := integerDecimal.ScanValue(reader, len(str))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, expectedBigInt, &integerDecimal.Int)
}

// Test that the negative big integers is deserialized from bytes.
func TestScanValueNegativeBigInt(t *testing.T) {
	// Arrange
	reader := &poolReaderMock{[]byte("-1"), nil}
	integerDecimal := newIntegerDecimalZero()

	// Act
	err := integerDecimal.ScanValue(reader, 2)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(-1), &integerDecimal.Int)
}

// Test that the empty buffer is not deserialized.
func TestScanValueEmptyBuffer(t *testing.T) {
	// Arrange
	reader := &poolReaderMock{[]byte(""), nil}
	integerDecimal := newIntegerDecimal(big.NewInt(42))

	// Act
	err := integerDecimal.ScanValue(reader, 0)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(0), &integerDecimal.Int)
}

// Test that the deserialization fails if error occurs.
func TestScanValueFailOnScannerError(t *testing.T) {
	// Arrange
	reader := &poolReaderMock{[]byte("foo"), errors.Errorf("bar")}
	integerDecimal := newIntegerDecimalZero()

	// Act
	err := integerDecimal.ScanValue(reader, 3)

	// Assert
	require.Error(t, err)
}
